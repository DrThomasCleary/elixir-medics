package csv_test

import (
	"fmt"
	"strings"

	"github.com/schani/elixir-medics/internal/csv"
	"github.com/schani/elixir-medics/internal/extract"
)

const utf8BOM = "\xEF\xBB\xBF"

func ExampleCSVWriter_WriteInvoice() {
	writer := csv.NewCSVWriter()

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
	fmt.Print(strings.TrimPrefix(result, utf8BOM))
	// Output:
	// Reference Number,Date of Referral,Referring GP,Date of Assessment,Type,Mode,Medication,Cost
	// REF001,2025-01-15,Dr. Smith,2025-01-20,Initial,Face-to-face,Yes,100.00
}

func ExampleCSVWriter_WriteAppointments() {
	writer := csv.NewCSVWriter()

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
	fmt.Print(strings.TrimPrefix(result, utf8BOM))
	// Output:
	// Patient Name,Referral ID,Referral Date,Appointment Date/Time,Arrived?
	// John Doe,REF001,2025-01-15,2025-01-20 10:00,Yes
}

func ExampleCSVWriter_WritePatients() {
	writer := csv.NewCSVWriter()

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
	fmt.Print(strings.TrimPrefix(result, utf8BOM))
	// Output:
	// Patient Name,Reference Number,Date of Referral,Referring GP,Mode,Medication,Discharge Date,Positive Diagnosis,Yearly Follow Up,Previous Diagnosis,Shared Care,Number of Appointments,Number of Treatment Notes,Number of Communications
	// John Doe,REF001,2025-01-15,Dr. Smith,Face-to-face,Yes,2025-02-15,Yes,2026-01-15,No,Yes,3,5,2
}

func ExampleCSVWriter_WriteSubmissions() {
	writer := csv.NewCSVWriter()

	r := extract.SubmissionsReport{
		CaseloadCount:                     50,
		DNACount:                          5,
		DNAPercentage:                     25.5,
		InitialAssessmentCount:            10,
		PsychologicalTherapiesCount:       8,
		NewDiagnosisCount:                 6,
		NewDiagnosisPercentage:            75.0,
		SharedCareCount:                   4,
		ReferralsCount:                    15,
		ReferralsWithPreviousDiagnosis:    3,
		ReferralsWithoutPreviousDiagnosis: 12,
		PatientContactsCount:              20,
	}

	result := writer.WriteSubmissions(r)
	fmt.Print(strings.TrimPrefix(result, utf8BOM))
	// Output:
	// Number of patients on caseload,50
	// Number of DNA contacts,5
	// Percentage DNA,25.5%
	// Number of patients who received initial assessment,10
	// Number of patients receiving psychological therapies,8
	// Number of patients diagnosed with ADHD with new diagnosis,6
	// Percentage of patients diagnosed with ADHD of new diagnosis referrals,75.0%
	// Number of patients prescribed medication under Shared Care,4
	// Number of Referrals,15
	// Referrals with previous diagnosis,3
	// Referrals without previous diagnosis,12
	// Number of patient contacts,20
}
