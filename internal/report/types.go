// Package report provides report generation functionality.
package report

import (
	"time"

	"github.com/schani/elixir-medics/internal/cliniko"
)

// Options configures report generation.
type Options struct {
	// OnPatientsFetched is called as patients are fetched from the API.
	// count is the number fetched so far, total is the final count (set when complete).
	OnPatientsFetched func(count int, total *int)

	// OnPatientsProcessed is called as patients are processed.
	OnPatientsProcessed func(processed, total int)

	// OnContactsFetched is called as contacts (treatment notes/communications) are fetched.
	OnContactsFetched func(processed, total int)

	// Month filters to only include rows from this month (1-12).
	Month *int

	// Year filters to only include rows from this year.
	Year *int

	// Now overrides the current time for testing. Defaults to time.Now().
	Now *time.Time
}

// Result contains the generated report data.
type Result struct {
	InvoiceCSV              string
	AppointmentsCSV         string
	MonthlyAppointmentsCSV  *string // Appointments filtered to the selected month
	PatientsCSV             string
	SubmissionsCSV          *string
	YearlyFollowUpCSV       string
	RawPatients             []cliniko.Patient
}
