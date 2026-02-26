// Package extract provides data extraction logic for patient records.
package extract

// TriState represents a Yes/No/N/A value.
type TriState string

const (
	TriStateYes TriState = "Yes"
	TriStateNo  TriState = "No"
	TriStateNA  TriState = "N/A"
)

// MedicationStatus represents the medication status: Yes (prescribed), Other (prescribed then stopped), No, or N/A.
type MedicationStatus string

const (
	MedStatusYes            MedicationStatus = "Yes"
	MedStatusOther          MedicationStatus = "Other"
	MedStatus10WeeksWaiting MedicationStatus = "10 Weeks Waiting"
	MedStatusNo             MedicationStatus = "No"
	MedStatusNA             MedicationStatus = "N/A"
)

// HasMedication returns true if the medication status is Yes or Other (both incur the £400 charge).
// "10 Weeks Waiting" does NOT count as having medication (patient is still waiting).
func (m MedicationStatus) HasMedication() bool {
	return m == MedStatusYes || m == MedStatusOther
}

// AppointmentMode represents the mode of an appointment.
type AppointmentMode string

const (
	ModeFaceToFace AppointmentMode = "Face-to-face"
	ModeRemote     AppointmentMode = "Remote"
	ModeNA         AppointmentMode = "N/A"
)

// AppointmentType represents the type of an appointment.
type AppointmentType string

const (
	TypeInitial  AppointmentType = "Initial"
	TypeFollowUp AppointmentType = "Follow-up"
	TypeNA       AppointmentType = "N/A"
)

// ParsedCustomFields contains parsed custom field values from a patient.
type ParsedCustomFields struct {
	ReferralDate      *string
	DischargeDate     *string
	Medication        MedicationStatus
	MedicationOther   string // Free-text value when medication status is "Other"
	PositiveDiagnosis *bool
	YearlyFollowUp    *string
	PreviousDiagnosis *bool
	SharedCare        *bool
}

// ExtractedRow represents a single row for the invoice/patients CSV.
type ExtractedRow struct {
	ReferenceNumber     string
	DateOfReferral      string
	ReferringGP         string
	DateOfAssessment    string
	DateOfAssessmentRaw string // ISO date for sorting/filtering
	Type                AppointmentType
	Mode                AppointmentMode
	Medication          MedicationStatus
	Cost                string // Cost in pounds, only for initial appointment
	DischargeDate       string
	PositiveDiagnosis   TriState
	YearlyFollowUp      string
	PreviousDiagnosis   TriState
	SharedCare          TriState
}

// AppointmentWithPatient represents an appointment enriched with patient info.
type AppointmentWithPatient struct {
	PatientName            string
	ReferralID             string
	ReferralDate           string
	PreviousDiagnosis      TriState
	AppointmentDateTime    string
	AppointmentDateTimeRaw string
	Arrived                bool
}

// PatientRow represents a summary row for the patients CSV.
type PatientRow struct {
	PatientName            string
	ReferenceNumber        string
	DateOfReferral         string
	ReferringGP            string
	Mode                   AppointmentMode
	Medication             MedicationStatus
	DischargeDate          string
	PositiveDiagnosis      TriState
	YearlyFollowUp         string
	PreviousDiagnosis      TriState
	SharedCare             TriState
	NumberOfAppointments   int
	NumberOfTreatmentNotes int
	NumberOfCommunications int
}

// YearlyFollowUpRow represents a patient due for yearly follow-up.
type YearlyFollowUpRow struct {
	PatientName     string
	ReferenceNumber string
	DischargeDate   string
	FollowUpDueDate string // Discharge date + 12 months
	Medication      MedicationStatus
	ReferringGP     string
}

// TenWeeksWaitingRow represents a patient waiting 10 weeks for medication.
type TenWeeksWaitingRow struct {
	PatientName     string
	ReferenceNumber string
	DateOfReferral  string
	ReferringGP     string
}

// PatientContact represents a treatment note or communication for statistics.
type PatientContact struct {
	CreatedAt string // ISO 8601
}

// ExtractionResult contains the results of extracting data for a patient.
type ExtractionResult struct {
	Rows         []ExtractedRow
	Appointments []AppointmentWithPatient
}

// SubmissionsReport contains statistics for the submissions report.
type SubmissionsReport struct {
	DNACount                          int
	DNAPercentage                     float64
	InitialAssessmentCount            int
	ReferralsCount                    int
	ReferralsWithPreviousDiagnosis    int
	ReferralsWithoutPreviousDiagnosis int
	PatientContactsCount              int
	// New fields
	CaseloadCount               int     // Patients referred but not discharged by end of month
	PsychologicalTherapiesCount int     // Patients with initial assessment in month + positive diagnosis
	NewDiagnosisCount           int     // Patients with positive diagnosis and no previous diagnosis (of those with initial assessment in month)
	NewDiagnosisPercentage      float64 // Percentage of new diagnosis patients (of those with initial assessment and no previous diagnosis)
	SharedCareCount             int     // Patients on caseload with medication and shared care
}
