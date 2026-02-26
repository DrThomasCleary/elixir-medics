package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements cliniko.Client for testing.
type mockClient struct {
	patients     []cliniko.Patient
	appointments map[int64][]cliniko.Appointment
	aptTypes     map[string]*cliniko.AppointmentType
	notes        map[int64][]cliniko.TreatmentNote
	comms        map[int64][]cliniko.Communication
}

func (m *mockClient) GetAllPatients(ctx context.Context) (<-chan cliniko.Patient, <-chan error) {
	patients := make(chan cliniko.Patient)
	errors := make(chan error, 1)

	go func() {
		defer close(patients)
		defer close(errors)
		for _, p := range m.patients {
			select {
			case <-ctx.Done():
				errors <- ctx.Err()
				return
			case patients <- p:
			}
		}
	}()

	return patients, errors
}

func (m *mockClient) GetAppointmentsForPatient(ctx context.Context, patientID int64) ([]cliniko.Appointment, error) {
	return m.appointments[patientID], nil
}

func (m *mockClient) GetAppointmentType(ctx context.Context, url string) (*cliniko.AppointmentType, error) {
	if apt, ok := m.aptTypes[url]; ok {
		return apt, nil
	}
	return &cliniko.AppointmentType{ID: 1, Name: "Face-to-face"}, nil
}

func (m *mockClient) GetTreatmentNotesForPatient(ctx context.Context, patientID int64) ([]cliniko.TreatmentNote, error) {
	return m.notes[patientID], nil
}

func (m *mockClient) GetCommunicationsForPatient(ctx context.Context, patientID int64) ([]cliniko.Communication, error) {
	return m.comms[patientID], nil
}

func ptr(s string) *string {
	return &s
}

func TestGenerator_Generate_EmptyPatients(t *testing.T) {
	client := &mockClient{
		patients: []cliniko.Patient{},
	}

	generator := NewGenerator(client)
	result, err := generator.Generate(context.Background(), Options{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.InvoiceCSV, "Reference Number")
	assert.Contains(t, result.AppointmentsCSV, "Patient Name")
	assert.Contains(t, result.PatientsCSV, "Reference Number")
	assert.Nil(t, result.SubmissionsCSV)
}

func TestGenerator_Generate_WithPatient(t *testing.T) {
	client := &mockClient{
		patients: []cliniko.Patient{
			{
				ID:             1,
				FirstName:      "John",
				LastName:       "Doe",
				OldReferenceID: ptr("EML1"),
			},
		},
		appointments: map[int64][]cliniko.Appointment{
			1: {
				{
					ID:               "apt1",
					AppointmentStart: "2024-10-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{Self: "https://api.cliniko.com/v1/appointment_types/1"},
					},
				},
			},
		},
		aptTypes: map[string]*cliniko.AppointmentType{
			"https://api.cliniko.com/v1/appointment_types/1": {
				ID:   1,
				Name: "Face-to-face Initial",
			},
		},
	}

	now := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	generator := NewGenerator(client)
	result, err := generator.Generate(context.Background(), Options{
		Now: &now,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.InvoiceCSV, "EML-C382005-2025-001")
	assert.Contains(t, result.InvoiceCSV, "Initial")
	assert.Contains(t, result.InvoiceCSV, "Face-to-face")
}

func TestGenerator_Generate_WithMonthFilter(t *testing.T) {
	client := &mockClient{
		patients: []cliniko.Patient{
			{
				ID:             1,
				FirstName:      "John",
				LastName:       "Doe",
				OldReferenceID: ptr("EML1"),
			},
		},
		appointments: map[int64][]cliniko.Appointment{
			1: {
				{
					ID:               "apt1",
					AppointmentStart: "2024-10-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{Self: "https://api.cliniko.com/v1/appointment_types/1"},
					},
				},
				{
					ID:               "apt2",
					AppointmentStart: "2024-11-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{Self: "https://api.cliniko.com/v1/appointment_types/1"},
					},
				},
			},
		},
	}

	now := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	month := 10
	year := 2024

	generator := NewGenerator(client)
	result, err := generator.Generate(context.Background(), Options{
		Now:   &now,
		Month: &month,
		Year:  &year,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Should only include October appointment
	assert.True(t, strings.Contains(result.InvoiceCSV, "15/10/2024"))
	assert.False(t, strings.Contains(result.InvoiceCSV, "15/11/2024"))
	// Should have submissions CSV when month/year filter is applied
	assert.NotNil(t, result.SubmissionsCSV)
}

func TestGenerator_Generate_SkipsPatientWithoutRefID(t *testing.T) {
	client := &mockClient{
		patients: []cliniko.Patient{
			{
				ID:             1,
				FirstName:      "John",
				LastName:       "Doe",
				OldReferenceID: nil, // No reference ID
			},
		},
	}

	generator := NewGenerator(client)
	result, err := generator.Generate(context.Background(), Options{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	// Invoice should only have header, no data rows
	// Count non-empty lines (CSV may have trailing newline)
	lines := strings.Split(strings.TrimSpace(result.InvoiceCSV), "\n")
	assert.Equal(t, 1, len(lines)) // Just header
}
