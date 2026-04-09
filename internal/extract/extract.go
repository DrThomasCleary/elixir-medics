package extract

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/schani/elixir-medics/internal/cliniko"
)

const (
	// NHS_ID is the standard NHS identifier used in reference numbers
	NHS_ID = "C382005-2025"
)

// FormatReferenceNumber formats a patient reference ID into the standard format.
// Example: "EML12" or "EML 13" -> "EML-C382005-2025-012" or "EML-C382005-2025-013"
func FormatReferenceNumber(refID string) string {
	// Extract the number from the reference
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(refID)
	if match == "" {
		return fmt.Sprintf("EML-%s-000", NHS_ID)
	}

	num, err := strconv.Atoi(match)
	if err != nil {
		return fmt.Sprintf("EML-%s-000", NHS_ID)
	}

	// Pad to 3 digits
	paddedNum := fmt.Sprintf("%03d", num)
	return fmt.Sprintf("EML-%s-%s", NHS_ID, paddedNum)
}

// FormatDateBritish formats an ISO 8601 date string to British format in the Europe/London timezone.
// Example: "2025-10-15T10:00:00Z" -> "15/10/2025, 10:00"
func FormatDateBritish(isoDate string) string {
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return "N/A"
	}

	// Load London timezone
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		loc = time.UTC
	}

	// Convert to London time
	tLondon := t.In(loc)

	// Format as dd/mm/yyyy, hh:mm
	return tLondon.Format("02/01/2006, 15:04")
}

// CalculateInitialCost returns the cost of an initial assessment.
// Face-to-face = 1025, remote = 925. Medication is billed separately as a titration row.
func CalculateInitialCost(isRemote bool) int {
	if isRemote {
		return 925
	}
	return 1025
}

// TitrationCost is the flat cost for a titration appointment.
const TitrationCost = 400

// BoolToTriState converts a bool pointer to a TriState value.
func BoolToTriState(b *bool) TriState {
	if b == nil {
		return TriStateNA
	}
	if *b {
		return TriStateYes
	}
	return TriStateNo
}

// stringOrNA returns the string value or "N/A" if nil.
func stringOrNA(s *string) string {
	if s == nil || *s == "" {
		return "N/A"
	}
	return *s
}

// ExtractPatientRows extracts invoice rows and appointment data for a single patient.
func ExtractPatientRows(
	ctx context.Context,
	client cliniko.Client,
	patient cliniko.Patient,
) (*ExtractionResult, error) {
	// Skip patients without old_reference_id
	if patient.OldReferenceID == nil || *patient.OldReferenceID == "" {
		return &ExtractionResult{
			Rows:         []ExtractedRow{},
			Appointments: []AppointmentWithPatient{},
		}, nil
	}

	// Fetch all appointments for this patient
	rawAppointments, err := client.GetAppointmentsForPatient(ctx, patient.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments for patient %d: %w", patient.ID, err)
	}

	// Parse custom fields for this patient
	customFields := ParseCustomFields(patient)

	// Transform appointments to include patient info
	patientName := fmt.Sprintf("%s %s", patient.FirstName, patient.LastName)
	referralID := *patient.OldReferenceID
	referralDate := stringOrNA(customFields.ReferralDate)
	previousDiagnosis := BoolToTriState(customFields.PreviousDiagnosis)

	appointmentsWithPatient := make([]AppointmentWithPatient, 0, len(rawAppointments))
	for _, apt := range rawAppointments {
		appointmentsWithPatient = append(appointmentsWithPatient, AppointmentWithPatient{
			PatientName:            patientName,
			ReferralID:             referralID,
			ReferralDate:           referralDate,
			PreviousDiagnosis:      previousDiagnosis,
			AppointmentDateTime:    FormatDateBritish(apt.AppointmentStart),
			AppointmentDateTimeRaw: apt.AppointmentStart,
			Arrived:                !apt.DidNotArrive,
		})
	}

	// Sort ALL appointments by date ascending
	allAppointmentsSorted := make([]cliniko.Appointment, len(rawAppointments))
	copy(allAppointmentsSorted, rawAppointments)
	sort.Slice(allAppointmentsSorted, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, allAppointmentsSorted[i].AppointmentStart)
		tj, _ := time.Parse(time.RFC3339, allAppointmentsSorted[j].AppointmentStart)
		return ti.Before(tj)
	})

	// Filter to appointments where patient showed up
	validAppointments := make([]cliniko.Appointment, 0)
	var referralCutoff *time.Time
	if customFields.ReferralDate != nil && *customFields.ReferralDate != "" {
		if rt, err := time.Parse("2006-01-02", *customFields.ReferralDate); err == nil {
			referralCutoff = &rt
		}
	}
	for _, apt := range allAppointmentsSorted {
		if !apt.DidNotArrive {
			if referralCutoff != nil {
				aptTime, err := time.Parse(time.RFC3339, apt.AppointmentStart)
				if err == nil && aptTime.Before(*referralCutoff) {
					// Ignore arrived appointments before NHS referral date
					// (typically private-clinic history not billable under EML rules).
					continue
				}
			}
			validAppointments = append(validAppointments, apt)
		}
	}

	// Take at most the first two valid appointments
	relevantAppointments := validAppointments
	if len(relevantAppointments) > 2 {
		relevantAppointments = relevantAppointments[:2]
	}

	// Medication from custom_fields
	// If "Other" is selected and has free-text, display as "Other - <text>"
	medication := customFields.Medication
	if medication == MedStatusOther && customFields.MedicationOther != "" {
		medication = MedicationStatus("Other - " + customFields.MedicationOther)
	}

	// New fields from custom_fields
	dischargeDate := stringOrNA(customFields.DischargeDate)
	positiveDiagnosis := BoolToTriState(customFields.PositiveDiagnosis)
	yearlyFollowUp := stringOrNA(customFields.YearlyFollowUp)
	sharedCare := BoolToTriState(customFields.SharedCare)

	rows := make([]ExtractedRow, 0)
	referenceNumber := FormatReferenceNumber(referralID)
	dateOfReferral := stringOrNA(customFields.ReferralDate)
	referringGP := ParseGP(patient.Notes)


	// If no valid appointments, still produce a row with N/A values
	if len(relevantAppointments) == 0 {
		rows = append(rows, ExtractedRow{
			ReferenceNumber:     referenceNumber,
			DateOfReferral:      dateOfReferral,
			ReferringGP:         referringGP,
			DateOfAssessment:    "N/A",
			DateOfAssessmentRaw: "",
			Type:                TypeNA,
			Mode:                ModeNA,
			Medication:          MedStatusNA,
			Cost:                "",
			DischargeDate:       dischargeDate,
			PositiveDiagnosis:   positiveDiagnosis,
			YearlyFollowUp:      yearlyFollowUp,
			PreviousDiagnosis:   previousDiagnosis,
			SharedCare:          sharedCare,
		})
		return &ExtractionResult{
			Rows:         rows,
			Appointments: appointmentsWithPatient,
		}, nil
	}

	// Mode is determined ONLY by the initial (first) appointment
	initialAppointmentTypeURL := relevantAppointments[0].AppointmentType.Links.Self
	isRemote := false
	initialAppointmentType, err := client.GetAppointmentType(ctx, initialAppointmentTypeURL)
	if err != nil {
		if cliniko.IsNotFound(err) {
			log.Printf("[WARN] Appointment type not found for patient %s (URL: %s), defaulting to Face-to-face", referenceNumber, initialAppointmentTypeURL)
		} else {
			return nil, fmt.Errorf("failed to get appointment type: %w", err)
		}
	} else {
		isRemote = strings.Contains(strings.ToLower(initialAppointmentType.Name), "remote")
	}
	mode := ModeFaceToFace
	if isRemote {
		mode = ModeRemote
	}

	// Initial assessment cost (medication is billed separately as a titration row)
	cost := fmt.Sprintf("£%d", CalculateInitialCost(isRemote))

	// Initial assessment row
	rows = append(rows, ExtractedRow{
		ReferenceNumber:     referenceNumber,
		DateOfReferral:      dateOfReferral,
		ReferringGP:         referringGP,
		DateOfAssessment:    FormatDateBritish(relevantAppointments[0].AppointmentStart),
		DateOfAssessmentRaw: relevantAppointments[0].AppointmentStart,
		Type:                TypeInitial,
		Mode:                mode,
		Medication:          medication,
		Cost:                cost,
		DischargeDate:       dischargeDate,
		PositiveDiagnosis:   positiveDiagnosis,
		YearlyFollowUp:      yearlyFollowUp,
		PreviousDiagnosis:   previousDiagnosis,
		SharedCare:          sharedCare,
	})

	// Second appointment: "Titration" if patient has medication, otherwise "Follow-up"
	if len(relevantAppointments) > 1 {
		secondType := TypeFollowUp
		secondCost := ""
		if medication.HasMedication() {
			secondType = TypeTitration
			secondCost = fmt.Sprintf("£%d", TitrationCost)
		}

		rows = append(rows, ExtractedRow{
			ReferenceNumber:     referenceNumber,
			DateOfReferral:      dateOfReferral,
			ReferringGP:         referringGP,
			DateOfAssessment:    FormatDateBritish(relevantAppointments[1].AppointmentStart),
			DateOfAssessmentRaw: relevantAppointments[1].AppointmentStart,
			Type:                secondType,
			Mode:                mode,
			Medication:          medication,
			Cost:                secondCost,
			DischargeDate:       dischargeDate,
			PositiveDiagnosis:   positiveDiagnosis,
			YearlyFollowUp:      yearlyFollowUp,
			PreviousDiagnosis:   previousDiagnosis,
			SharedCare:          sharedCare,
		})
	}

	return &ExtractionResult{
		Rows:         rows,
		Appointments: appointmentsWithPatient,
	}, nil
}
