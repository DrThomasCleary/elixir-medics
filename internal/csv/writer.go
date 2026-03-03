// Package csv provides CSV generation functionality.
package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/schani/elixir-medics/internal/extract"
)

const utf8BOM = "\xEF\xBB\xBF"

// formatDateUK converts a YYYY-MM-DD date to DD/MM/YYYY for display.
// Returns the original string if it can't be parsed (e.g. "N/A").
func formatDateUK(isoDate string) string {
	t, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("02/01/2006")
}

// CSVWriter implements the Writer interface using Go's standard library.
type CSVWriter struct{}

// NewCSVWriter creates a new CSV writer.
func NewCSVWriter() Writer {
	return &CSVWriter{}
}

// WriteInvoice generates the invoice CSV.
func (w *CSVWriter) WriteInvoice(rows []extract.ExtractedRow) string {
	headers := []string{
		"Reference Number",
		"Date of Referral",
		"Referring GP",
		"Date of Assessment",
		"Type",
		"Mode",
		"Medication",
		"Cost",
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)

	for _, row := range rows {
		_ = writer.Write([]string{
			row.ReferenceNumber,
			formatDateUK(row.DateOfReferral),
			row.ReferringGP,
			row.DateOfAssessment,
			string(row.Type),
			string(row.Mode),
			string(row.Medication),
			row.Cost,
		})
	}

	writer.Flush()
	return buf.String()
}

// WriteAppointments generates the appointments CSV.
func (w *CSVWriter) WriteAppointments(appointments []extract.AppointmentWithPatient) string {
	headers := []string{
		"Patient Name",
		"Referral ID",
		"Referral Date",
		"Appointment Date/Time",
		"Arrived?",
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)

	for _, apt := range appointments {
		arrived := "No"
		if apt.Arrived {
			arrived = "Yes"
		}

		_ = writer.Write([]string{
			apt.PatientName,
			apt.ReferralID,
			formatDateUK(apt.ReferralDate),
			apt.AppointmentDateTime,
			arrived,
		})
	}

	writer.Flush()
	return buf.String()
}

// WritePatients generates the patients CSV.
func (w *CSVWriter) WritePatients(patients []extract.PatientRow) string {
	headers := []string{
		"Patient Name",
		"Reference Number",
		"Date of Referral",
		"Referring GP",
		"Mode",
		"Medication",
		"Discharge Date",
		"Positive Diagnosis",
		"Yearly Follow Up",
		"Previous Diagnosis",
		"Shared Care",
		"Number of Appointments",
		"Number of Treatment Notes",
		"Number of Communications",
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)

	for _, patient := range patients {
		_ = writer.Write([]string{
			patient.PatientName,
			patient.ReferenceNumber,
			formatDateUK(patient.DateOfReferral),
			patient.ReferringGP,
			string(patient.Mode),
			string(patient.Medication),
			formatDateUK(patient.DischargeDate),
			string(patient.PositiveDiagnosis),
			patient.YearlyFollowUp,
			string(patient.PreviousDiagnosis),
			string(patient.SharedCare),
			strconv.Itoa(patient.NumberOfAppointments),
			strconv.Itoa(patient.NumberOfTreatmentNotes),
			strconv.Itoa(patient.NumberOfCommunications),
		})
	}

	writer.Flush()
	return buf.String()
}

// WriteSubmissions generates the submissions report CSV.
// Format: key-value pairs without headers.
func (w *CSVWriter) WriteSubmissions(r extract.SubmissionsReport) string {
	lines := []string{
		fmt.Sprintf("Number of patients on caseload,%d", r.CaseloadCount),
		fmt.Sprintf("Number of DNA contacts,%d", r.DNACount),
		fmt.Sprintf("Percentage DNA,%.1f%%", r.DNAPercentage),
		fmt.Sprintf("Number of patients who received initial assessment,%d", r.InitialAssessmentCount),
		fmt.Sprintf("Number of patients receiving psychological therapies,%d", r.PsychologicalTherapiesCount),
		fmt.Sprintf("Number of patients diagnosed with ADHD with new diagnosis,%d", r.NewDiagnosisCount),
		fmt.Sprintf("Percentage of patients diagnosed with ADHD of new diagnosis referrals,%.1f%%", r.NewDiagnosisPercentage),
		fmt.Sprintf("Number of patients prescribed medication under Shared Care,%d", r.SharedCareCount),
		fmt.Sprintf("Number of Referrals,%d", r.ReferralsCount),
		fmt.Sprintf("Referrals with previous diagnosis,%d", r.ReferralsWithPreviousDiagnosis),
		fmt.Sprintf("Referrals without previous diagnosis,%d", r.ReferralsWithoutPreviousDiagnosis),
		fmt.Sprintf("Number of patient contacts,%d", r.PatientContactsCount),
		fmt.Sprintf("New assessments (face-to-face),%d", r.InitialFaceToFaceCount),
		fmt.Sprintf("New assessments (remote),%d", r.InitialRemoteCount),
		fmt.Sprintf("Number of titration appointments,%d", r.TitrationCount),
	}

	result := utf8BOM
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// WriteTenWeeksWaiting generates the 10 weeks waiting list CSV.
func (w *CSVWriter) WriteTenWeeksWaiting(rows []extract.TenWeeksWaitingRow) string {
	headers := []string{
		"Patient Name",
		"Reference Number",
		"Date of Referral",
		"Referring GP",
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)

	for _, row := range rows {
		_ = writer.Write([]string{
			row.PatientName,
			row.ReferenceNumber,
			formatDateUK(row.DateOfReferral),
			row.ReferringGP,
		})
	}

	writer.Flush()
	return buf.String()
}

// WriteYearlyFollowUp generates the yearly follow-up list CSV.
func (w *CSVWriter) WriteYearlyFollowUp(rows []extract.YearlyFollowUpRow) string {
	headers := []string{
		"Patient Name",
		"Reference Number",
		"Discharge Date",
		"Follow-Up Due Date",
		"Medication",
		"Referring GP",
	}

	var buf bytes.Buffer
	buf.WriteString(utf8BOM)
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)

	for _, row := range rows {
		_ = writer.Write([]string{
			row.PatientName,
			row.ReferenceNumber,
			formatDateUK(row.DischargeDate),
			row.FollowUpDueDate,
			string(row.Medication),
			row.ReferringGP,
		})
	}

	writer.Flush()
	return buf.String()
}
