package csv

import (
	"strings"
	"testing"

	"github.com/schani/elixir-medics/internal/extract"
	
)

func TestCSVWriter_WriteInvoice(t *testing.T) {
	writer := NewCSVWriter()

	t.Run("generates correct CSV with headers", func(t *testing.T) {
		rows := []extract.ExtractedRow{
			{
				ReferenceNumber:  "REF001",
				DateOfReferral:   "2025-01-15",
				ReferringGP:      "Dr. Smith",
				DateOfAssessment: "2025-01-20",
				Type:             extract.TypeInitial,
				Mode:             extract.ModeFaceToFace,
				Medication:       extract.MedStatusYes,
				Cost:             "100.00",
			},
		}

		result := writer.WriteInvoice(rows)

		expectedLines := []string{
			"Reference Number,Date of Referral,Referring GP,Date of Assessment,Type,Mode,Medication,Cost",
			"REF001,2025-01-15,Dr. Smith,2025-01-20,Initial,Face-to-face,Yes,100.00",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Errorf("Expected CSV to contain %q, got:\n%s", line, result)
			}
		}
	})

	t.Run("handles empty rows", func(t *testing.T) {
		rows := []extract.ExtractedRow{}
		result := writer.WriteInvoice(rows)

		expected := "Reference Number,Date of Referral,Referring GP,Date of Assessment,Type,Mode,Medication,Cost"
		if !strings.Contains(result, expected) {
			t.Errorf("Expected header line, got: %s", result)
		}
	})

	t.Run("escapes values with commas", func(t *testing.T) {
		rows := []extract.ExtractedRow{
			{
				ReferenceNumber:  "REF001",
				DateOfReferral:   "2025-01-15",
				ReferringGP:      "Smith, John",
				DateOfAssessment: "2025-01-20",
				Type:             extract.TypeInitial,
				Mode:             extract.ModeFaceToFace,
				Medication:       extract.MedStatusYes,
				Cost:             "100.00",
			},
		}

		result := writer.WriteInvoice(rows)

		if !strings.Contains(result, `"Smith, John"`) {
			t.Errorf("Expected comma-containing value to be quoted, got:\n%s", result)
		}
	})

	t.Run("escapes values with quotes", func(t *testing.T) {
		rows := []extract.ExtractedRow{
			{
				ReferenceNumber:  "REF001",
				DateOfReferral:   "2025-01-15",
				ReferringGP:      `Dr. "Bob" Smith`,
				DateOfAssessment: "2025-01-20",
				Type:             extract.TypeInitial,
				Mode:             extract.ModeFaceToFace,
				Medication:       extract.MedStatusYes,
				Cost:             "100.00",
			},
		}

		result := writer.WriteInvoice(rows)

		if !strings.Contains(result, `"Dr. ""Bob"" Smith"`) {
			t.Errorf("Expected quotes to be doubled and value quoted, got:\n%s", result)
		}
	})

	t.Run("escapes values with newlines", func(t *testing.T) {
		rows := []extract.ExtractedRow{
			{
				ReferenceNumber:  "REF001",
				DateOfReferral:   "2025-01-15",
				ReferringGP:      "Dr. Smith\nClinic",
				DateOfAssessment: "2025-01-20",
				Type:             extract.TypeInitial,
				Mode:             extract.ModeFaceToFace,
				Medication:       extract.MedStatusYes,
				Cost:             "100.00",
			},
		}

		result := writer.WriteInvoice(rows)

		if !strings.Contains(result, `"Dr. Smith`) {
			t.Errorf("Expected newline-containing value to be quoted, got:\n%s", result)
		}
	})

	t.Run("handles multiple rows", func(t *testing.T) {
		rows := []extract.ExtractedRow{
			{
				ReferenceNumber:  "REF001",
				DateOfReferral:   "2025-01-15",
				ReferringGP:      "Dr. Smith",
				DateOfAssessment: "2025-01-20",
				Type:             extract.TypeInitial,
				Mode:             extract.ModeFaceToFace,
				Medication:       extract.MedStatusYes,
				Cost:             "100.00",
			},
			{
				ReferenceNumber:  "REF002",
				DateOfReferral:   "2025-01-16",
				ReferringGP:      "Dr. Jones",
				DateOfAssessment: "2025-01-21",
				Type:             extract.TypeFollowUp,
				Mode:             extract.ModeRemote,
				Medication:       extract.MedStatusNo,
				Cost:             "",
			},
		}

		result := writer.WriteInvoice(rows)
		lines := strings.Split(strings.TrimSpace(result), "\n")

		if len(lines) != 3 {
			t.Errorf("Expected 3 lines (header + 2 rows), got %d", len(lines))
		}
	})
}

func TestCSVWriter_WriteAppointments(t *testing.T) {
	writer := NewCSVWriter()

	t.Run("generates correct CSV with headers", func(t *testing.T) {
		appointments := []extract.AppointmentWithPatient{
			{
				PatientName:         "John Doe",
				ReferralID:          "REF001",
				ReferralDate:        "2025-01-15",
				AppointmentDateTime: "2025-01-20 10:00",
				Arrived:             true,
			},
		}

		result := writer.WriteAppointments(appointments)

		expectedLines := []string{
			"Patient Name,Referral ID,Referral Date,Appointment Date/Time,Arrived?",
			"John Doe,REF001,2025-01-15,2025-01-20 10:00,Yes",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Errorf("Expected CSV to contain %q, got:\n%s", line, result)
			}
		}
	})

	t.Run("handles arrived false", func(t *testing.T) {
		appointments := []extract.AppointmentWithPatient{
			{
				PatientName:         "John Doe",
				ReferralID:          "REF001",
				ReferralDate:        "2025-01-15",
				AppointmentDateTime: "2025-01-20 10:00",
				Arrived:             false,
			},
		}

		result := writer.WriteAppointments(appointments)

		if !strings.Contains(result, ",No") {
			t.Errorf("Expected Arrived to be 'No', got:\n%s", result)
		}
	})

	t.Run("escapes patient names with commas", func(t *testing.T) {
		appointments := []extract.AppointmentWithPatient{
			{
				PatientName:         "Doe, John",
				ReferralID:          "REF001",
				ReferralDate:        "2025-01-15",
				AppointmentDateTime: "2025-01-20 10:00",
				Arrived:             true,
			},
		}

		result := writer.WriteAppointments(appointments)

		if !strings.Contains(result, `"Doe, John"`) {
			t.Errorf("Expected comma-containing name to be quoted, got:\n%s", result)
		}
	})
}

func TestCSVWriter_WritePatients(t *testing.T) {
	writer := NewCSVWriter()

	t.Run("generates correct CSV with headers", func(t *testing.T) {
		patients := []extract.PatientRow{
			{
				PatientName:            "John Doe",
				ReferenceNumber:        "REF001",
				DateOfReferral:         "2025-01-15",
				ReferringGP:            "Dr. Smith",
				Mode:                   extract.ModeFaceToFace,
				Medication:             extract.MedStatusYes,
				DischargeDate:          "2025-02-15",
				PositiveDiagnosis:      extract.TriStateYes,
				YearlyFollowUp:         "2026-01-15",
				PreviousDiagnosis:      extract.TriStateNo,
				SharedCare:             extract.TriStateYes,
				NumberOfAppointments:   3,
				NumberOfTreatmentNotes: 5,
				NumberOfCommunications: 2,
			},
		}

		result := writer.WritePatients(patients)

		expectedLines := []string{
			"Patient Name,Reference Number,Date of Referral,Referring GP,Mode,Medication,Discharge Date,Positive Diagnosis,Yearly Follow Up,Previous Diagnosis,Shared Care,Number of Appointments,Number of Treatment Notes,Number of Communications",
			"John Doe,REF001,2025-01-15,Dr. Smith,Face-to-face,Yes,2025-02-15,Yes,2026-01-15,No,Yes,3,5,2",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Errorf("Expected CSV to contain %q, got:\n%s", line, result)
			}
		}
	})

	t.Run("converts integers to strings", func(t *testing.T) {
		patients := []extract.PatientRow{
			{
				PatientName:            "Jane Smith",
				ReferenceNumber:        "REF001",
				DateOfReferral:         "2025-01-15",
				ReferringGP:            "Dr. Smith",
				Mode:                   extract.ModeFaceToFace,
				Medication:             extract.MedStatusYes,
				DischargeDate:          "",
				PositiveDiagnosis:      extract.TriStateNA,
				YearlyFollowUp:         "",
				PreviousDiagnosis:      extract.TriStateNA,
				SharedCare:             extract.TriStateNA,
				NumberOfAppointments:   10,
				NumberOfTreatmentNotes: 20,
				NumberOfCommunications: 30,
			},
		}

		result := writer.WritePatients(patients)

		if !strings.Contains(result, ",10,20,30") {
			t.Errorf("Expected integer values to be correctly converted, got:\n%s", result)
		}
	})
}

func TestCSVWriter_WriteSubmissions(t *testing.T) {
	writer := NewCSVWriter()

	t.Run("generates correct key-value format with all fields", func(t *testing.T) {
		report := extract.SubmissionsReport{
			DNACount:                          5,
			DNAPercentage:                     25.5,
			InitialAssessmentCount:            10,
			ReferralsCount:                    15,
			ReferralsWithPreviousDiagnosis:    3,
			ReferralsWithoutPreviousDiagnosis: 12,
			PatientContactsCount:              20,
			CaseloadCount:                     50,
			PsychologicalTherapiesCount:       8,
			NewDiagnosisCount:                 6,
			NewDiagnosisPercentage:            75.0,
			SharedCareCount:                   4,
		}

		result := writer.WriteSubmissions(report)

		expectedLines := []string{
			"Number of patients on caseload,50",
			"Number of DNA contacts,5", // Note: renamed from "Number of DNA"
			"Percentage DNA,25.5%",
			"Number of patients who received initial assessment,10",
			"Number of patients receiving psychological therapies,8",
			"Number of patients diagnosed with ADHD with new diagnosis,6",
			"Percentage of patients diagnosed with ADHD of new diagnosis referrals,75.0%",
			"Number of patients prescribed medication under Shared Care,4",
			"Number of Referrals,15",
			"Referrals with previous diagnosis,3",
			"Referrals without previous diagnosis,12",
			"Number of patient contacts,20",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Errorf("Expected submissions report to contain %q, got:\n%s", line, result)
			}
		}
	})

	t.Run("formats percentage with one decimal place", func(t *testing.T) {
		report := extract.SubmissionsReport{
			DNACount:               1,
			DNAPercentage:          33.333333,
			NewDiagnosisPercentage: 66.666666,
		}

		result := writer.WriteSubmissions(report)

		if !strings.Contains(result, "Percentage DNA,33.3%") {
			t.Errorf("Expected DNA percentage to be formatted with 1 decimal place, got:\n%s", result)
		}
		if !strings.Contains(result, "new diagnosis referrals,66.7%") {
			t.Errorf("Expected new diagnosis percentage to be formatted with 1 decimal place, got:\n%s", result)
		}
	})

	t.Run("handles zero values", func(t *testing.T) {
		report := extract.SubmissionsReport{
			DNACount:                          0,
			DNAPercentage:                     0.0,
			InitialAssessmentCount:            0,
			ReferralsCount:                    0,
			ReferralsWithPreviousDiagnosis:    0,
			ReferralsWithoutPreviousDiagnosis: 0,
			PatientContactsCount:              0,
			CaseloadCount:                     0,
			PsychologicalTherapiesCount:       0,
			NewDiagnosisCount:                 0,
			NewDiagnosisPercentage:            0.0,
			SharedCareCount:                   0,
		}

		result := writer.WriteSubmissions(report)

		expectedLines := []string{
			"Number of patients on caseload,0",
			"Number of DNA contacts,0",
			"Percentage DNA,0.0%",
			"Number of patients who received initial assessment,0",
			"Number of patients receiving psychological therapies,0",
			"Number of patients diagnosed with ADHD with new diagnosis,0",
			"Percentage of patients diagnosed with ADHD of new diagnosis referrals,0.0%",
			"Number of patients prescribed medication under Shared Care,0",
			"Number of Referrals,0",
			"Referrals with previous diagnosis,0",
			"Referrals without previous diagnosis,0",
			"Number of patient contacts,0",
		}

		for _, line := range expectedLines {
			if !strings.Contains(result, line) {
				t.Errorf("Expected submissions report to contain %q, got:\n%s", line, result)
			}
		}
	})

	t.Run("has no trailing newline", func(t *testing.T) {
		report := extract.SubmissionsReport{
			DNACount:                          5,
			DNAPercentage:                     25.5,
			InitialAssessmentCount:            10,
			ReferralsCount:                    15,
			ReferralsWithPreviousDiagnosis:    3,
			ReferralsWithoutPreviousDiagnosis: 12,
			PatientContactsCount:              20,
			CaseloadCount:                     50,
			PsychologicalTherapiesCount:       8,
			NewDiagnosisCount:                 6,
			NewDiagnosisPercentage:            75.0,
			SharedCareCount:                   4,
		}

		result := writer.WriteSubmissions(report)

		if strings.HasSuffix(result, "\n\n") {
			t.Errorf("Expected no trailing newline, got extra newlines")
		}
	})

	t.Run("caseload count is first line", func(t *testing.T) {
		report := extract.SubmissionsReport{
			CaseloadCount: 42,
		}

		result := writer.WriteSubmissions(report)
		result = strings.TrimPrefix(result, utf8BOM)
		lines := strings.Split(result, "\n")

		if !strings.HasPrefix(lines[0], "Number of patients on caseload,42") {
			t.Errorf("Expected caseload to be first line, got: %s", lines[0])
		}
	})
}
