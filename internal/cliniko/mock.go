package cliniko

import "context"

// MockClient is a mock implementation of the Client interface for testing.
type MockClient struct {
	GetAllPatientsFunc              func(ctx context.Context) (<-chan Patient, <-chan error)
	GetAppointmentsForPatientFunc   func(ctx context.Context, patientID int64) ([]Appointment, error)
	GetAppointmentTypeFunc          func(ctx context.Context, url string) (*AppointmentType, error)
	GetTreatmentNotesForPatientFunc func(ctx context.Context, patientID int64) ([]TreatmentNote, error)
	GetCommunicationsForPatientFunc func(ctx context.Context, patientID int64) ([]Communication, error)
}

// GetAllPatients calls the mock function if set, otherwise returns empty channels.
func (m *MockClient) GetAllPatients(ctx context.Context) (<-chan Patient, <-chan error) {
	if m.GetAllPatientsFunc != nil {
		return m.GetAllPatientsFunc(ctx)
	}
	patientsCh := make(chan Patient)
	errCh := make(chan error)
	close(patientsCh)
	close(errCh)
	return patientsCh, errCh
}

// GetAppointmentsForPatient calls the mock function if set, otherwise returns empty slice.
func (m *MockClient) GetAppointmentsForPatient(ctx context.Context, patientID int64) ([]Appointment, error) {
	if m.GetAppointmentsForPatientFunc != nil {
		return m.GetAppointmentsForPatientFunc(ctx, patientID)
	}
	return []Appointment{}, nil
}

// GetAppointmentType calls the mock function if set, otherwise returns nil.
func (m *MockClient) GetAppointmentType(ctx context.Context, url string) (*AppointmentType, error) {
	if m.GetAppointmentTypeFunc != nil {
		return m.GetAppointmentTypeFunc(ctx, url)
	}
	return nil, nil
}

// GetTreatmentNotesForPatient calls the mock function if set, otherwise returns empty slice.
func (m *MockClient) GetTreatmentNotesForPatient(ctx context.Context, patientID int64) ([]TreatmentNote, error) {
	if m.GetTreatmentNotesForPatientFunc != nil {
		return m.GetTreatmentNotesForPatientFunc(ctx, patientID)
	}
	return []TreatmentNote{}, nil
}

// GetCommunicationsForPatient calls the mock function if set, otherwise returns empty slice.
func (m *MockClient) GetCommunicationsForPatient(ctx context.Context, patientID int64) ([]Communication, error) {
	if m.GetCommunicationsForPatientFunc != nil {
		return m.GetCommunicationsForPatientFunc(ctx, patientID)
	}
	return []Communication{}, nil
}
