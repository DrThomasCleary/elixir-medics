package extract

import (
	"strings"

	"github.com/schani/elixir-medics/internal/cliniko"
)

// Custom field tokens (matched by token, not name, as names can change)
const (
	TokenReferralDate         = "4d1cf1ed-d346-468a-8e0d-23e9525e866b"
	TokenDischargeDate        = "2b8a8cce-4a6f-4ab1-8deb-4dd8cac7cd7b"
	TokenDiagnosed            = "720db7f7-a3bf-45c4-833c-fea5bac6cdab"
	TokenDiagnosedPositive    = "7eceed68-fff9-4dc4-a06b-8042ba690953"
	TokenMedication           = "d152c6e3-b2ce-44ff-843c-3ef0825519c6"
	TokenMedicationPrescribed = "4f7da0a8-a112-40aa-8727-63046c788799"
	TokenYearlyFollowUp       = "9466f420-4f30-422b-a750-159739730fc7"
	TokenPreviousDiagnosis    = "330f7335-b6a2-4fb1-bbfe-81c31c0f0936"
	TokenPreviousDiagnosisYes = "979f3616-9777-4feb-b149-4eca93a85ff9"
	TokenSharedCare           = "01e24511-3203-4451-b72e-b5428794c39d"
	TokenSharedCareYes        = "3010f410-76e5-4d53-9812-16a5fe0bf266"
)

// ParseCustomFields extracts relevant custom field values from a patient record.
func ParseCustomFields(patient cliniko.Patient) ParsedCustomFields {
	result := ParsedCustomFields{
		ReferralDate:      nil,
		DischargeDate:     nil,
		Medication:        MedStatusNA,
		PositiveDiagnosis: nil,
		YearlyFollowUp:    nil,
		PreviousDiagnosis: nil,
		SharedCare:        nil,
	}

	if patient.CustomFields == nil {
		return result
	}

	// Flatten all fields from all sections
	var allFields []cliniko.CustomField
	for _, section := range patient.CustomFields.Sections {
		allFields = append(allFields, section.Fields...)
	}

	for _, field := range allFields {
		switch field.Token {
		case TokenReferralDate:
			result.ReferralDate = field.Value
		case TokenDischargeDate:
			result.DischargeDate = field.Value
		case TokenDiagnosed:
			// Check if the "Positive" option is selected
			positiveDiagnosis := false
			for _, opt := range field.Options {
				if opt.Token == TokenDiagnosedPositive && opt.Selected {
					positiveDiagnosis = true
					break
				}
			}
			result.PositiveDiagnosis = &positiveDiagnosis
		case TokenMedication:
			// Check medication options:
			// "Prescribed" → Yes, "10 weeks" in name → 10 Weeks Waiting, any other → Other, none → No
			prescribed := false
			tenWeeksWaiting := false
			otherSelected := false
			otherBody := ""
			for _, opt := range field.Options {
				if opt.Selected {
					if opt.Token == TokenMedicationPrescribed {
						prescribed = true
					} else if strings.Contains(strings.ToLower(opt.Name), "10 week") {
						tenWeeksWaiting = true
					} else {
						body := ""
						if opt.Body != nil && *opt.Body != "" {
							body = *opt.Body
						} else {
							body = opt.Name
						}
						if strings.Contains(strings.ToLower(body), "not prescribed") {
							// "Not Prescribed" variants → no charge
						} else {
							otherSelected = true
							otherBody = body
						}
					}
				}
			}
			if prescribed {
				result.Medication = MedStatusYes
			} else if tenWeeksWaiting {
				result.Medication = MedStatus10WeeksWaiting
			} else if otherSelected {
				result.Medication = MedStatusOther
				result.MedicationOther = otherBody
			} else {
				result.Medication = MedStatusNo
			}
		case TokenYearlyFollowUp:
			// Get the name of the selected option
			for _, opt := range field.Options {
				if opt.Selected {
					result.YearlyFollowUp = &opt.Name
					break
				}
			}
		case TokenPreviousDiagnosis:
			// Check if the "Yes" option is selected
			previousDiagnosis := false
			for _, opt := range field.Options {
				if opt.Token == TokenPreviousDiagnosisYes && opt.Selected {
					previousDiagnosis = true
					break
				}
			}
			result.PreviousDiagnosis = &previousDiagnosis
		case TokenSharedCare:
			// Check if the "Yes" option is selected
			sharedCare := false
			for _, opt := range field.Options {
				if opt.Token == TokenSharedCareYes && opt.Selected {
					sharedCare = true
					break
				}
			}
			result.SharedCare = &sharedCare
		}
	}

	return result
}
