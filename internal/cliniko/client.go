package cliniko

import "context"

// Client defines the interface for Cliniko API operations.
type Client interface {
	// GetAllPatients returns all patients, yielding them through channels.
	// The patients channel receives patients one by one.
	// The errors channel receives any error that occurs (and closes both channels).
	// Both channels are closed when iteration is complete.
	GetAllPatients(ctx context.Context) (<-chan Patient, <-chan error)

	// GetAppointmentsForPatient fetches all appointments for a patient.
	GetAppointmentsForPatient(ctx context.Context, patientID int64) ([]Appointment, error)

	// GetAppointmentType fetches appointment type details by URL.
	GetAppointmentType(ctx context.Context, url string) (*AppointmentType, error)

	// GetTreatmentNotesForPatient fetches all treatment notes for a patient.
	GetTreatmentNotesForPatient(ctx context.Context, patientID int64) ([]TreatmentNote, error)

	// GetCommunicationsForPatient fetches all communications for a patient.
	GetCommunicationsForPatient(ctx context.Context, patientID int64) ([]Communication, error)
}
