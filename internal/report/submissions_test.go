package report

import (
	"testing"

	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/schani/elixir-medics/internal/extract"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSubmissionsReport_DNACount(t *testing.T) {
	appointments := []extract.AppointmentWithPatient{
		{ReferralID: "EML1", AppointmentDateTimeRaw: "2024-11-15T10:00:00Z", Arrived: true},
		{ReferralID: "EML2", AppointmentDateTimeRaw: "2024-11-16T10:00:00Z", Arrived: false}, // DNA
		{ReferralID: "EML3", AppointmentDateTimeRaw: "2024-11-17T10:00:00Z", Arrived: false}, // DNA
		{ReferralID: "EML4", AppointmentDateTimeRaw: "2024-10-15T10:00:00Z", Arrived: false}, // DNA but wrong month
	}

	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-10-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-11-15T10:00:00Z"},
		{ReferenceNumber: "EML-002", DateOfReferral: "2024-10-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-11-16T10:00:00Z"},
		{ReferenceNumber: "EML-003", DateOfReferral: "2024-10-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-11-17T10:00:00Z"},
	}

	report := GenerateSubmissionsReport(appointments, nil, rows, nil, 11, 2024)

	assert.Equal(t, 2, report.DNACount, "Should count 2 DNA appointments in November")
	// DNA=2, initial assessments=1, DNA% = 2/(2+1) = 66.7%
	assert.InDelta(t, 66.67, report.DNAPercentage, 0.1)
}

func TestGenerateSubmissionsReport_InitialAssessmentCount(t *testing.T) {
	// Patient 1: first arrived appointment in November
	// Patient 2: first arrived appointment in October
	// Patient 3: has appointment in November but it's DNA, first arrived is in December
	appointments := []extract.AppointmentWithPatient{
		// Patient 1: first appointment in Nov (arrived)
		{ReferralID: "EML1", AppointmentDateTimeRaw: "2024-11-15T10:00:00Z", Arrived: true},
		{ReferralID: "EML1", AppointmentDateTimeRaw: "2024-12-15T10:00:00Z", Arrived: true},
		// Patient 2: first appointment in Oct
		{ReferralID: "EML2", AppointmentDateTimeRaw: "2024-10-15T10:00:00Z", Arrived: true},
		{ReferralID: "EML2", AppointmentDateTimeRaw: "2024-11-15T10:00:00Z", Arrived: true},
		// Patient 3: first appointment in Nov is DNA, first arrived in Dec
		{ReferralID: "EML3", AppointmentDateTimeRaw: "2024-11-10T10:00:00Z", Arrived: false},
		{ReferralID: "EML3", AppointmentDateTimeRaw: "2024-12-10T10:00:00Z", Arrived: true},
	}

	report := GenerateSubmissionsReport(appointments, nil, nil, nil, 11, 2024)

	// Only Patient 1 has their first arrived appointment in November
	assert.Equal(t, 1, report.InitialAssessmentCount)
}

func TestGenerateSubmissionsReport_ReferralsCount(t *testing.T) {
	refDate1 := "2024-11-01"
	refDate2 := "2024-11-15"
	refDate3 := "2024-10-15"
	patients := []cliniko.Patient{
		{ID: 1, FirstName: "A", LastName: "A", CustomFields: &cliniko.CustomFields{Sections: []cliniko.CustomFieldSection{{Fields: []cliniko.CustomField{
			{Token: extract.TokenReferralDate, Value: &refDate1},
		}}}}},
		{ID: 2, FirstName: "B", LastName: "B", CustomFields: &cliniko.CustomFields{Sections: []cliniko.CustomFieldSection{{Fields: []cliniko.CustomField{
			{Token: extract.TokenReferralDate, Value: &refDate2},
			{Token: extract.TokenPreviousDiagnosis, Options: []cliniko.CustomFieldOption{{Token: extract.TokenPreviousDiagnosisYes, Selected: true}}},
		}}}}},
		{ID: 3, FirstName: "C", LastName: "C", CustomFields: &cliniko.CustomFields{Sections: []cliniko.CustomFieldSection{{Fields: []cliniko.CustomField{
			{Token: extract.TokenReferralDate, Value: &refDate3},
		}}}}},
	}

	report := GenerateSubmissionsReport(nil, nil, nil, patients, 11, 2024)

	assert.Equal(t, 2, report.ReferralsCount, "Should count 2 referrals in November")
	assert.Equal(t, 1, report.ReferralsWithPreviousDiagnosis)
	assert.Equal(t, 1, report.ReferralsWithoutPreviousDiagnosis)
}

func TestGenerateSubmissionsReport_PatientContactsCount(t *testing.T) {
	contacts := []extract.PatientContact{
		{CreatedAt: "2024-11-01T10:00:00Z"},
		{CreatedAt: "2024-11-15T10:00:00Z"},
		{CreatedAt: "2024-10-15T10:00:00Z"}, // Wrong month
		{CreatedAt: "2024-11-30T23:59:00Z"},
	}

	report := GenerateSubmissionsReport(nil, contacts, nil, nil, 11, 2024)

	assert.Equal(t, 3, report.PatientContactsCount, "Should count 3 contacts in November")
}

func TestGenerateSubmissionsReport_CaseloadCount(t *testing.T) {
	// Reporting month: November 2024 (end of month: 2024-11-30)
	// Case 1: Referred in October, not discharged -> ON caseload
	// Case 2: Referred in November, not discharged -> ON caseload
	// Case 3: Referred in October, discharged in October -> NOT on caseload
	// Case 4: Referred in October, discharged in November -> NOT on caseload
	// Case 5: Referred in October, discharged in December -> ON caseload
	// Case 6: Referred in December -> NOT on caseload (after end of month)

	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-10-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z"},
		{ReferenceNumber: "EML-002", DateOfReferral: "2024-11-15", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-11-20T10:00:00Z"},
		{ReferenceNumber: "EML-003", DateOfReferral: "2024-10-01", DischargeDate: "2024-10-31", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z"},
		{ReferenceNumber: "EML-004", DateOfReferral: "2024-10-01", DischargeDate: "2024-11-15", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z"},
		{ReferenceNumber: "EML-005", DateOfReferral: "2024-10-01", DischargeDate: "2024-12-01", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z"},
		{ReferenceNumber: "EML-006", DateOfReferral: "2024-12-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-12-10T10:00:00Z"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	// EML-001, EML-002, EML-005 should be on caseload
	assert.Equal(t, 3, report.CaseloadCount, "Should count 3 patients on caseload")
}

func TestGenerateSubmissionsReport_CaseloadCount_DischargedOnLastDay(t *testing.T) {
	// Edge case: discharged on the last day of the month
	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-10-01", DischargeDate: "2024-11-30", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	// Discharged on end of month - should NOT be on caseload
	assert.Equal(t, 0, report.CaseloadCount)
}

func TestGenerateSubmissionsReport_PsychologicalTherapiesCount(t *testing.T) {
	// Patients with initial assessment in reporting month AND positive diagnosis
	rows := []extract.ExtractedRow{
		// Initial in Nov, positive diagnosis -> count
		{ReferenceNumber: "EML-001", DateOfAssessmentRaw: "2024-11-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
		// Initial in Nov, negative diagnosis -> don't count
		{ReferenceNumber: "EML-002", DateOfAssessmentRaw: "2024-11-16T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateNo, DateOfReferral: "2024-10-01"},
		// Initial in Nov, N/A diagnosis -> don't count
		{ReferenceNumber: "EML-003", DateOfAssessmentRaw: "2024-11-17T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateNA, DateOfReferral: "2024-10-01"},
		// Initial in Oct, positive diagnosis -> don't count (wrong month)
		{ReferenceNumber: "EML-004", DateOfAssessmentRaw: "2024-10-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
		// Initial in Nov, positive diagnosis -> count
		{ReferenceNumber: "EML-005", DateOfAssessmentRaw: "2024-11-20T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	assert.Equal(t, 2, report.PsychologicalTherapiesCount, "Should count 2 patients receiving psychological therapies")
}

func TestGenerateSubmissionsReport_NewDiagnosisCount(t *testing.T) {
	appointments := []extract.AppointmentWithPatient{
		{ReferralID: "EML1", AppointmentDateTimeRaw: "2024-11-15T10:00:00Z", Arrived: true},
		{ReferralID: "EML2", AppointmentDateTimeRaw: "2024-11-16T10:00:00Z", Arrived: true},
		{ReferralID: "EML3", AppointmentDateTimeRaw: "2024-11-17T10:00:00Z", Arrived: true},
		{ReferralID: "EML4", AppointmentDateTimeRaw: "2024-11-18T10:00:00Z", Arrived: true},
		{ReferralID: "EML5", AppointmentDateTimeRaw: "2024-10-15T10:00:00Z", Arrived: true},
	}

	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfAssessmentRaw: "2024-11-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNo, DateOfReferral: "2024-10-01"},
		{ReferenceNumber: "EML-002", DateOfAssessmentRaw: "2024-11-16T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateNo, PreviousDiagnosis: extract.TriStateNo, DateOfReferral: "2024-10-01"},
		{ReferenceNumber: "EML-003", DateOfAssessmentRaw: "2024-11-17T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
		{ReferenceNumber: "EML-004", DateOfAssessmentRaw: "2024-11-18T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNA, DateOfReferral: "2024-10-01"},
		{ReferenceNumber: "EML-005", DateOfAssessmentRaw: "2024-10-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNo, DateOfReferral: "2024-10-01"},
	}

	report := GenerateSubmissionsReport(appointments, nil, rows, nil, 11, 2024)

	assert.Equal(t, 2, report.NewDiagnosisCount, "Should count 2 new diagnoses")
	// 2 new diagnoses / 4 initial assessments in Nov = 50%
	assert.InDelta(t, 50.0, report.NewDiagnosisPercentage, 0.1)
}

func TestGenerateSubmissionsReport_NewDiagnosisPercentage_ZeroEligible(t *testing.T) {
	// Edge case: all patients have previous diagnosis
	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfAssessmentRaw: "2024-11-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	assert.Equal(t, 0, report.NewDiagnosisCount)
	assert.Equal(t, 0.0, report.NewDiagnosisPercentage, "Should be 0% when no eligible patients")
}

func TestGenerateSubmissionsReport_EmptyInputs(t *testing.T) {
	report := GenerateSubmissionsReport(nil, nil, nil, nil, 11, 2024)

	assert.Equal(t, 0, report.DNACount)
	assert.Equal(t, 0.0, report.DNAPercentage)
	assert.Equal(t, 0, report.InitialAssessmentCount)
	assert.Equal(t, 0, report.ReferralsCount)
	assert.Equal(t, 0, report.PatientContactsCount)
	assert.Equal(t, 0, report.CaseloadCount)
	assert.Equal(t, 0, report.PsychologicalTherapiesCount)
	assert.Equal(t, 0, report.NewDiagnosisCount)
	assert.Equal(t, 0.0, report.NewDiagnosisPercentage)
}

func TestGenerateSubmissionsReport_FollowUpRowsIgnored(t *testing.T) {
	// Only Initial type rows should be considered for patient data
	rows := []extract.ExtractedRow{
		// Initial row for patient - should be used
		{ReferenceNumber: "EML-001", DateOfAssessmentRaw: "2024-11-15T10:00:00Z", Type: extract.TypeInitial, PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNo, DateOfReferral: "2024-10-01"},
		// Follow-up row for same patient - should be ignored for patient data
		{ReferenceNumber: "EML-001", DateOfAssessmentRaw: "2024-11-20T10:00:00Z", Type: extract.TypeFollowUp, PositiveDiagnosis: extract.TriStateNo, PreviousDiagnosis: extract.TriStateYes, DateOfReferral: "2024-10-01"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	// Should use Initial row data, not Follow-up
	assert.Equal(t, 1, report.PsychologicalTherapiesCount, "Should use Initial row diagnosis")
	assert.Equal(t, 1, report.NewDiagnosisCount, "Should use Initial row for new diagnosis")
}

func TestGenerateSubmissionsReport_CaseloadIncludesOldReferrals(t *testing.T) {
	// Important: caseload should include patients referred BEFORE the start of the reporting month
	rows := []extract.ExtractedRow{
		// Referred in January, not discharged -> ON caseload in November
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-01-15", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-01-20T10:00:00Z"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	assert.Equal(t, 1, report.CaseloadCount, "Should include patients referred before report month")
}

func TestGenerateSubmissionsReport_CaseloadReferredOnLastDay(t *testing.T) {
	// Edge case: referred on the last day of the reporting month
	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-11-30", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-12-05T10:00:00Z"},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 11, 2024)

	assert.Equal(t, 1, report.CaseloadCount, "Should include patient referred on last day of month")
}

func TestGenerateSubmissionsReport_December(t *testing.T) {
	// Test December specifically (month 12) to verify end-of-month calculation
	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-12-15", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-12-20T10:00:00Z", PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNo},
	}

	report := GenerateSubmissionsReport(nil, nil, rows, nil, 12, 2024)

	assert.Equal(t, 1, report.CaseloadCount)
	assert.Equal(t, 1, report.PsychologicalTherapiesCount)
	assert.Equal(t, 1, report.NewDiagnosisCount)
}

func TestGenerateSubmissionsReport_CompleteScenario(t *testing.T) {
	// Comprehensive test with all features
	appointments := []extract.AppointmentWithPatient{
		{ReferralID: "EML1", ReferralDate: "2024-11-01", AppointmentDateTimeRaw: "2024-11-10T10:00:00Z", Arrived: true, PreviousDiagnosis: extract.TriStateNo},
		{ReferralID: "EML1", ReferralDate: "2024-11-01", AppointmentDateTimeRaw: "2024-11-15T10:00:00Z", Arrived: true, PreviousDiagnosis: extract.TriStateNo},
		{ReferralID: "EML2", ReferralDate: "2024-10-01", AppointmentDateTimeRaw: "2024-11-20T10:00:00Z", Arrived: false, PreviousDiagnosis: extract.TriStateYes}, // DNA
	}

	contacts := []extract.PatientContact{
		{CreatedAt: "2024-11-05T10:00:00Z"},
		{CreatedAt: "2024-11-15T10:00:00Z"},
	}

	rows := []extract.ExtractedRow{
		{ReferenceNumber: "EML-001", DateOfReferral: "2024-11-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-11-10T10:00:00Z", PositiveDiagnosis: extract.TriStateYes, PreviousDiagnosis: extract.TriStateNo},
		{ReferenceNumber: "EML-002", DateOfReferral: "2024-10-01", DischargeDate: "N/A", Type: extract.TypeInitial, DateOfAssessmentRaw: "2024-10-15T10:00:00Z", PositiveDiagnosis: extract.TriStateNo, PreviousDiagnosis: extract.TriStateYes},
	}

	refDateNov := "2024-11-01"
	refDateOct := "2024-10-01"
	scenarioPatients := []cliniko.Patient{
		{ID: 1, FirstName: "A", LastName: "A", CustomFields: &cliniko.CustomFields{Sections: []cliniko.CustomFieldSection{{Fields: []cliniko.CustomField{
			{Token: extract.TokenReferralDate, Value: &refDateNov},
		}}}}},
		{ID: 2, FirstName: "B", LastName: "B", CustomFields: &cliniko.CustomFields{Sections: []cliniko.CustomFieldSection{{Fields: []cliniko.CustomField{
			{Token: extract.TokenReferralDate, Value: &refDateOct},
		}}}}},
	}

	report := GenerateSubmissionsReport(appointments, contacts, rows, scenarioPatients, 11, 2024)

	assert.Equal(t, 1, report.DNACount)
	assert.InDelta(t, 50.0, report.DNAPercentage, 0.1) // 1 DNA / (1 DNA + 1 initial) = 50%
	assert.Equal(t, 1, report.InitialAssessmentCount)   // EML1's first arrived in Nov
	assert.Equal(t, 1, report.ReferralsCount)           // Only patient 1 referred in Nov
	assert.Equal(t, 2, report.PatientContactsCount)
	assert.Equal(t, 2, report.CaseloadCount) // Both on caseload
	assert.Equal(t, 1, report.PsychologicalTherapiesCount)
	assert.Equal(t, 1, report.NewDiagnosisCount)
	assert.Equal(t, 100.0, report.NewDiagnosisPercentage) // 1 new diagnosis / 1 initial assessment
}
