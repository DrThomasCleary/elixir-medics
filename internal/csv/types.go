// Package csv provides CSV generation functionality.
package csv

import (
	"github.com/schani/elixir-medics/internal/extract"
)

// Writer defines the interface for CSV generation.
type Writer interface {
	// WriteInvoice generates the invoice CSV.
	WriteInvoice(rows []extract.ExtractedRow) string

	// WriteAppointments generates the appointments CSV.
	WriteAppointments(appointments []extract.AppointmentWithPatient) string

	// WritePatients generates the patients CSV.
	WritePatients(patients []extract.PatientRow) string

	// WriteSubmissions generates the submissions report CSV.
	WriteSubmissions(report extract.SubmissionsReport) string

	// WriteYearlyFollowUp generates the yearly follow-up list CSV.
	WriteYearlyFollowUp(rows []extract.YearlyFollowUpRow) string
}
