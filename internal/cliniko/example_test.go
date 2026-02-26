package cliniko_test

import (
	"context"
	"fmt"
	"log"

	"github.com/schani/elixir-medics/internal/cliniko"
)

// ExampleNewClient demonstrates how to create a new Cliniko client.
func ExampleNewClient() {
	// Create a new client with your API key
	client := cliniko.NewClient("your-api-key-here")

	// Use the client
	ctx := context.Background()
	patientsCh, errCh := client.GetAllPatients(ctx)

	// Process patients
	for patient := range patientsCh {
		fmt.Printf("Patient: %s %s\n", patient.FirstName, patient.LastName)
	}

	// Check for errors
	if err := <-errCh; err != nil {
		log.Fatal(err)
	}
}

// ExampleClientImpl_GetAllPatients demonstrates how to fetch all patients.
func ExampleClientImpl_GetAllPatients() {
	client := cliniko.NewClient("your-api-key-here")
	ctx := context.Background()

	// GetAllPatients returns two channels: one for patients, one for errors
	patientsCh, errCh := client.GetAllPatients(ctx)

	// Process patients as they arrive
	for patient := range patientsCh {
		fmt.Printf("Patient ID: %d, Name: %s %s\n",
			patient.ID, patient.FirstName, patient.LastName)
	}

	// Check for errors after all patients are processed
	if err := <-errCh; err != nil {
		log.Printf("Error fetching patients: %v", err)
	}
}

// ExampleClientImpl_GetAppointmentsForPatient demonstrates how to fetch appointments.
func ExampleClientImpl_GetAppointmentsForPatient() {
	client := cliniko.NewClient("your-api-key-here")
	ctx := context.Background()

	var patientID int64 = 12345
	appointments, err := client.GetAppointmentsForPatient(ctx, patientID)
	if err != nil {
		log.Fatal(err)
	}

	for _, apt := range appointments {
		fmt.Printf("Appointment ID: %s, Start: %s\n",
			apt.ID, apt.AppointmentStart)
	}
}

// ExampleClientImpl_GetAppointmentType demonstrates how to fetch appointment type details.
func ExampleClientImpl_GetAppointmentType() {
	client := cliniko.NewClient("your-api-key-here")
	ctx := context.Background()

	url := "https://api.uk2.cliniko.com/v1/appointment_types/123"
	aptType, err := client.GetAppointmentType(ctx, url)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Appointment Type: %s (ID: %d)\n", aptType.Name, aptType.ID)
}

// ExampleMockClient demonstrates how to use the mock client for testing.
func ExampleMockClient() {
	// Create a mock client
	mock := &cliniko.MockClient{
		GetAppointmentsForPatientFunc: func(ctx context.Context, patientID int64) ([]cliniko.Appointment, error) {
			// Return mock data
			return []cliniko.Appointment{
				{ID: "1", AppointmentStart: "2024-01-01T10:00:00Z"},
				{ID: "2", AppointmentStart: "2024-01-02T11:00:00Z"},
			}, nil
		},
	}

	// Use the mock in your tests
	ctx := context.Background()
	appointments, err := mock.GetAppointmentsForPatient(ctx, 123)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d appointments\n", len(appointments))
	// Output: Found 2 appointments
}
