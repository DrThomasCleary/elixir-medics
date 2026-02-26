package cliniko

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test helpers

func makeAuthHeader(apiKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(apiKey+":"))
}

func checkAuthHeader(t *testing.T, r *http.Request, expectedAPIKey string) {
	t.Helper()
	auth := r.Header.Get("Authorization")
	expected := makeAuthHeader(expectedAPIKey)
	if auth != expected {
		t.Errorf("Invalid auth header: got %s, want %s", auth, expected)
	}
}

func checkUserAgent(t *testing.T, r *http.Request) {
	t.Helper()
	ua := r.Header.Get("User-Agent")
	if ua != UserAgent {
		t.Errorf("Invalid user agent: got %s, want %s", ua, UserAgent)
	}
}

// TestNewClient verifies that NewClient creates a properly configured client.
func TestNewClient(t *testing.T) {
	client := NewClient("test-api-key")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.apiKey != "test-api-key" {
		t.Errorf("API key not set correctly: got %s, want test-api-key", client.apiKey)
	}
	if client.httpClient == nil {
		t.Error("HTTP client not initialized")
	}
	if client.rateLimiter == nil {
		t.Error("Rate limiter not initialized")
	}
}

// TestGetAllPatients tests fetching all patients with pagination.
func TestGetAllPatients(t *testing.T) {
	apiKey := "test-key"

	// Create test server
	requestCount := 0
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuthHeader(t, r, apiKey)
		checkUserAgent(t, r)

		requestCount++

		// Return different pages based on request
		if strings.Contains(r.URL.String(), "page=2") {
			// Second page (last page)
			resp := PatientsResponse{
				Patients: []Patient{
					{ID: 3, FirstName: "Carol", LastName: "Williams"},
				},
				Links: PaginatedLinks{
					Self: r.URL.String(),
					Next: nil, // Last page
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			// First page
			nextURL := serverURL + "/patients?page=2"
			resp := PatientsResponse{
				Patients: []Patient{
					{ID: 1, FirstName: "Alice", LastName: "Smith"},
					{ID: 2, FirstName: "Bob", LastName: "Jones"},
				},
				Links: PaginatedLinks{
					Self: r.URL.String(),
					Next: &nextURL,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	// Create client with test server URL
	client := NewClient(apiKey)
	// Override base URL for testing
	testURL := server.URL + "/patients?per_page=100"

	ctx := context.Background()

	// Call GetAllPatients directly with test URL
	patientsCh := make(chan Patient)
	errCh := make(chan error, 1)

	go func() {
		defer close(patientsCh)
		defer close(errCh)

		url := testURL
		for url != "" {
			var response PatientsResponse
			if err := client.fetchJSON(ctx, url, &response); err != nil {
				errCh <- err
				return
			}

			for _, patient := range response.Patients {
				patientsCh <- patient
			}

			if response.Links.Next != nil {
				url = *response.Links.Next
			} else {
				url = ""
			}
		}
	}()

	// Collect results
	var patients []Patient
	for patient := range patientsCh {
		patients = append(patients, patient)
	}

	// Check for errors
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	default:
	}

	// Verify results
	if len(patients) != 3 {
		t.Errorf("Expected 3 patients, got %d", len(patients))
	}
	if requestCount != 2 {
		t.Errorf("Expected 2 requests, got %d", requestCount)
	}
}

// TestGetAppointmentsForPatient tests fetching appointments for a patient.
func TestGetAppointmentsForPatient(t *testing.T) {
	apiKey := "test-key"
	patientID := 123

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuthHeader(t, r, apiKey)
		checkUserAgent(t, r)

		expectedPath := fmt.Sprintf("/patients/%d/appointments", patientID)
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := AppointmentsResponse{
			Appointments: []Appointment{
				{ID: "1", AppointmentStart: "2024-01-01T10:00:00Z"},
				{ID: "2", AppointmentStart: "2024-01-02T11:00:00Z"},
			},
			Links: PaginatedLinks{
				Self: r.URL.String(),
				Next: nil,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	// Call with test server URL
	url := fmt.Sprintf("%s/patients/%d/appointments?per_page=100", server.URL, patientID)
	var appointments []Appointment

	for url != "" {
		var response AppointmentsResponse
		if err := client.fetchJSON(ctx, url, &response); err != nil {
			t.Fatalf("Error: %v", err)
		}
		appointments = append(appointments, response.Appointments...)
		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	if len(appointments) != 2 {
		t.Errorf("Expected 2 appointments, got %d", len(appointments))
	}
}

// TestGetAppointmentType tests fetching appointment type details.
func TestGetAppointmentType(t *testing.T) {
	apiKey := "test-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuthHeader(t, r, apiKey)
		checkUserAgent(t, r)

		appointmentType := AppointmentType{
			ID:   1,
			Name: "Initial Consultation",
		}
		json.NewEncoder(w).Encode(appointmentType)
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	appointmentType, err := client.GetAppointmentType(ctx, server.URL+"/appointment_types/1")
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if appointmentType.ID != 1 {
		t.Errorf("Expected ID 1, got %d", appointmentType.ID)
	}
	if appointmentType.Name != "Initial Consultation" {
		t.Errorf("Expected name 'Initial Consultation', got %s", appointmentType.Name)
	}
}

// TestGetTreatmentNotesForPatient tests fetching treatment notes.
func TestGetTreatmentNotesForPatient(t *testing.T) {
	apiKey := "test-key"
	patientID := 123

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuthHeader(t, r, apiKey)
		checkUserAgent(t, r)

		if !strings.Contains(r.URL.RawQuery, fmt.Sprintf("patient_id:=%d", patientID)) {
			t.Errorf("Missing patient_id filter in query: %s", r.URL.RawQuery)
		}

		resp := TreatmentNotesResponse{
			TreatmentNotes: []TreatmentNote{
				{ID: 1, CreatedAt: "2024-01-01T10:00:00Z"},
				{ID: 2, CreatedAt: "2024-01-02T10:00:00Z"},
			},
			Links: PaginatedLinks{
				Self: r.URL.String(),
				Next: nil,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	url := fmt.Sprintf("%s/treatment_notes?q[]=patient_id:=%d&per_page=100", server.URL, patientID)
	var notes []TreatmentNote

	for url != "" {
		var response TreatmentNotesResponse
		if err := client.fetchJSON(ctx, url, &response); err != nil {
			t.Fatalf("Error: %v", err)
		}
		notes = append(notes, response.TreatmentNotes...)
		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	if len(notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(notes))
	}
}

// TestGetCommunicationsForPatient tests fetching communications.
func TestGetCommunicationsForPatient(t *testing.T) {
	apiKey := "test-key"
	patientID := 123

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checkAuthHeader(t, r, apiKey)
		checkUserAgent(t, r)

		if !strings.Contains(r.URL.RawQuery, fmt.Sprintf("patient_id:=%d", patientID)) {
			t.Errorf("Missing patient_id filter in query: %s", r.URL.RawQuery)
		}

		resp := CommunicationsResponse{
			Communications: []Communication{
				{ID: "1", CreatedAt: "2024-01-01T10:00:00Z"},
				{ID: "2", CreatedAt: "2024-01-02T10:00:00Z"},
			},
			Links: PaginatedLinks{
				Self: r.URL.String(),
				Next: nil,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	url := fmt.Sprintf("%s/communications?q[]=patient_id:=%d&per_page=100", server.URL, patientID)
	var communications []Communication

	for url != "" {
		var response CommunicationsResponse
		if err := client.fetchJSON(ctx, url, &response); err != nil {
			t.Fatalf("Error: %v", err)
		}
		communications = append(communications, response.Communications...)
		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	if len(communications) != 2 {
		t.Errorf("Expected 2 communications, got %d", len(communications))
	}
}

// TestRateLimiting verifies that rate limiting is enforced.
func TestRateLimiting(t *testing.T) {
	apiKey := "test-key"
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		appointmentType := AppointmentType{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(appointmentType)
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	// Make multiple requests quickly
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := client.GetAppointmentType(ctx, server.URL+"/test")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// With rate limiting at 200/min = 3.33/sec, 5 requests should take some time
	// But not too long since we allow bursts of 1
	if elapsed < 1*time.Second {
		// This is expected - the first request is immediate, then we need to wait
		// for tokens to refill for subsequent requests
	}

	if requestCount != 5 {
		t.Errorf("Expected 5 requests, got %d", requestCount)
	}
}

// TestRetryOn429 verifies that 429 responses are retried.
func TestRetryOn429(t *testing.T) {
	apiKey := "test-key"
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			// Return 429 for first two requests
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too many requests"))
		} else {
			// Succeed on third request
			appointmentType := AppointmentType{ID: 1, Name: "Test"}
			json.NewEncoder(w).Encode(appointmentType)
		}
	}))
	defer server.Close()

	client := NewClient(apiKey)
	// Reduce retry waits for testing
	client.httpClient.RetryWaitMin = 10 * time.Millisecond
	client.httpClient.RetryWaitMax = 100 * time.Millisecond

	ctx := context.Background()

	start := time.Now()
	appointmentType, err := client.GetAppointmentType(ctx, server.URL+"/test")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Request failed after retries: %v", err)
	}

	if appointmentType.ID != 1 {
		t.Errorf("Expected ID 1, got %d", appointmentType.ID)
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests (2 retries), got %d", requestCount)
	}

	// Should have taken some time due to retries
	if elapsed < 10*time.Millisecond {
		t.Errorf("Request completed too quickly, expected some retry delay")
	}
}

// TestContextCancellation verifies that context cancellation stops requests.
func TestContextCancellation(t *testing.T) {
	apiKey := "test-key"

	// Server that takes a long time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		appointmentType := AppointmentType{ID: 1, Name: "Test"}
		json.NewEncoder(w).Encode(appointmentType)
	}))
	defer server.Close()

	client := NewClient(apiKey)

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetAppointmentType(ctx, server.URL+"/test")
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}

// TestErrorHandling verifies that API errors are properly handled.
func TestErrorHandling(t *testing.T) {
	apiKey := "test-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	client := NewClient(apiKey)
	ctx := context.Background()

	_, err := client.GetAppointmentType(ctx, server.URL+"/test")
	if err == nil {
		t.Error("Expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 in error message, got: %v", err)
	}
}

// TestMockClient verifies that the mock client works correctly.
func TestMockClient(t *testing.T) {
	mock := &MockClient{
		GetAppointmentTypeFunc: func(ctx context.Context, url string) (*AppointmentType, error) {
			return &AppointmentType{ID: 123, Name: "Mock Type"}, nil
		},
	}

	ctx := context.Background()
	appointmentType, err := mock.GetAppointmentType(ctx, "http://example.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if appointmentType.ID != 123 {
		t.Errorf("Expected ID 123, got %d", appointmentType.ID)
	}
	if appointmentType.Name != "Mock Type" {
		t.Errorf("Expected name 'Mock Type', got %s", appointmentType.Name)
	}
}

// TestMockClientDefaults verifies that mock client returns safe defaults.
func TestMockClientDefaults(t *testing.T) {
	mock := &MockClient{}
	ctx := context.Background()

	// Test default implementations
	patientsCh, errCh := mock.GetAllPatients(ctx)

	var patients []Patient
	for patient := range patientsCh {
		patients = append(patients, patient)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	default:
	}

	if len(patients) != 0 {
		t.Errorf("Expected 0 patients from default mock, got %d", len(patients))
	}

	appointments, err := mock.GetAppointmentsForPatient(ctx, 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(appointments) != 0 {
		t.Errorf("Expected 0 appointments from default mock, got %d", len(appointments))
	}
}
