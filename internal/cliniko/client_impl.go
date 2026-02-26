package cliniko

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/time/rate"
)

const (
	// APIBaseURL is the base URL for the Cliniko UK API
	APIBaseURL = "https://api.uk2.cliniko.com/v1"
	// UserAgent is the user agent string sent with requests
	UserAgent = "Elixir Medics (mark.probst@gmail.com)"
	// RequestsPerMinute is the rate limit for API requests
	RequestsPerMinute = 200
	// PerPage is the number of items to request per page
	PerPage = 100
)

// ClientImpl implements the Client interface for the Cliniko API.
type ClientImpl struct {
	apiKey      string
	httpClient  *retryablehttp.Client
	rateLimiter *rate.Limiter
}

// NewClient creates a new Cliniko API client with rate limiting and retry logic.
func NewClient(apiKey string) *ClientImpl {
	// Create rate limiter: 200 requests per minute = 200/60 per second
	rateLimiter := rate.NewLimiter(rate.Limit(RequestsPerMinute/60.0), 1)

	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 60 * time.Second
	retryClient.CheckRetry = checkRetry
	retryClient.Logger = nil // Disable logging

	return &ClientImpl{
		apiKey:      apiKey,
		httpClient:  retryClient,
		rateLimiter: rateLimiter,
	}
}

// checkRetry determines if a request should be retried.
// Returns true for 429 (Too Many Requests) and standard retryable errors.
func checkRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// Check context first
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	// If there's an error, use default retry logic
	if err != nil {
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	// Always retry on 429 (Too Many Requests)
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}

	// Use default retry policy for other cases
	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// doRequest performs an authenticated HTTP request with rate limiting.
func (c *ClientImpl) doRequest(ctx context.Context, url string) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Create request
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	auth := base64.StdEncoding.EncodeToString([]byte(c.apiKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil, &NotFoundError{URL: url, Body: string(body)}
		}
		return nil, fmt.Errorf("API error: %d %s: %s", resp.StatusCode, resp.Status, string(body))
	}

	return resp, nil
}

// fetchJSON performs a request and decodes the JSON response.
func (c *ClientImpl) fetchJSON(ctx context.Context, url string, v interface{}) error {
	resp, err := c.doRequest(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// GetAllPatients returns all patients, yielding them through channels.
// The patients channel receives patients one by one.
// The errors channel receives any error that occurs (and closes both channels).
// Both channels are closed when iteration is complete.
func (c *ClientImpl) GetAllPatients(ctx context.Context) (<-chan Patient, <-chan error) {
	patientsCh := make(chan Patient)
	errCh := make(chan error, 1)

	go func() {
		defer close(patientsCh)
		defer close(errCh)

		url := fmt.Sprintf("%s/patients?per_page=%d", APIBaseURL, PerPage)

		for url != "" {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			var response PatientsResponse
			if err := c.fetchJSON(ctx, url, &response); err != nil {
				errCh <- err
				return
			}

			for _, patient := range response.Patients {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case patientsCh <- patient:
				}
			}

			if response.Links.Next != nil {
				url = *response.Links.Next
			} else {
				url = ""
			}
		}
	}()

	return patientsCh, errCh
}

// GetAppointmentsForPatient fetches all appointments for a patient.
func (c *ClientImpl) GetAppointmentsForPatient(ctx context.Context, patientID int64) ([]Appointment, error) {
	var appointments []Appointment
	url := fmt.Sprintf("%s/patients/%d/appointments?per_page=%d", APIBaseURL, patientID, PerPage)

	for url != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var response AppointmentsResponse
		if err := c.fetchJSON(ctx, url, &response); err != nil {
			return nil, err
		}

		appointments = append(appointments, response.Appointments...)

		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	return appointments, nil
}

// GetAppointmentType fetches appointment type details by URL.
func (c *ClientImpl) GetAppointmentType(ctx context.Context, url string) (*AppointmentType, error) {
	var appointmentType AppointmentType
	if err := c.fetchJSON(ctx, url, &appointmentType); err != nil {
		return nil, err
	}
	return &appointmentType, nil
}

// GetTreatmentNotesForPatient fetches all treatment notes for a patient.
func (c *ClientImpl) GetTreatmentNotesForPatient(ctx context.Context, patientID int64) ([]TreatmentNote, error) {
	var notes []TreatmentNote
	url := fmt.Sprintf("%s/treatment_notes?q[]=patient_id:=%d&per_page=%d", APIBaseURL, patientID, PerPage)

	for url != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var response TreatmentNotesResponse
		if err := c.fetchJSON(ctx, url, &response); err != nil {
			return nil, err
		}

		notes = append(notes, response.TreatmentNotes...)

		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	return notes, nil
}

// GetCommunicationsForPatient fetches all communications for a patient.
func (c *ClientImpl) GetCommunicationsForPatient(ctx context.Context, patientID int64) ([]Communication, error) {
	var communications []Communication
	url := fmt.Sprintf("%s/communications?q[]=patient_id:=%d&per_page=%d", APIBaseURL, patientID, PerPage)

	for url != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var response CommunicationsResponse
		if err := c.fetchJSON(ctx, url, &response); err != nil {
			return nil, err
		}

		communications = append(communications, response.Communications...)

		if response.Links.Next != nil {
			url = *response.Links.Next
		} else {
			url = ""
		}
	}

	return communications, nil
}
