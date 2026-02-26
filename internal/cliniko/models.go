// Package cliniko provides a client for the Cliniko healthcare API.
package cliniko

import "fmt"

// NotFoundError is returned when the API responds with 404.
type NotFoundError struct {
	URL  string
	Body string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s: %s", e.URL, e.Body)
}

// IsNotFound returns true if the error is a 404 Not Found from the API.
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// CustomFieldOption represents an option in a custom field.
type CustomFieldOption struct {
	Name     string  `json:"name"`
	Token    string  `json:"token"`
	Selected bool    `json:"selected,omitempty"`
	Body     *string `json:"body,omitempty"`
}

// CustomField represents a custom field in a patient record.
type CustomField struct {
	Name    string              `json:"name"`
	Type    string              `json:"type"`
	Token   string              `json:"token"`
	Value   *string             `json:"value,omitempty"`
	Options []CustomFieldOption `json:"options,omitempty"`
}

// CustomFieldSection represents a section of custom fields.
type CustomFieldSection struct {
	Name   string        `json:"name"`
	Token  string        `json:"token"`
	Fields []CustomField `json:"fields"`
}

// CustomFields represents all custom fields for a patient.
type CustomFields struct {
	Sections []CustomFieldSection `json:"sections"`
}

// PatientLinks contains links for a patient.
type PatientLinks struct {
	Self string `json:"self"`
}

// Patient represents a patient from the Cliniko API.
type Patient struct {
	ID             int64         `json:"id,string"`
	FirstName      string        `json:"first_name"`
	LastName       string        `json:"last_name"`
	OldReferenceID *string       `json:"old_reference_id"`
	ReferralSource *string       `json:"referral_source"`
	Notes          *string       `json:"notes"`
	CustomFields   *CustomFields `json:"custom_fields"`
	Links          PatientLinks  `json:"links"`
}

// AppointmentTypeLink contains a link to an appointment type.
type AppointmentTypeLink struct {
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
}

// AppointmentLinks contains links for an appointment.
type AppointmentLinks struct {
	Self string `json:"self"`
}

// Appointment represents an appointment from the Cliniko API.
type Appointment struct {
	ID               string              `json:"id"`
	AppointmentStart string              `json:"appointment_start"`
	DidNotArrive     bool                `json:"did_not_arrive"`
	AppointmentType  AppointmentTypeLink `json:"appointment_type"`
	Links            AppointmentLinks    `json:"links"`
}

// AppointmentType represents an appointment type from the Cliniko API.
type AppointmentType struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}

// TreatmentNote represents a treatment note from the Cliniko API.
type TreatmentNote struct {
	ID        int64  `json:"id,string"`
	CreatedAt string `json:"created_at"`
}

// Communication represents a communication from the Cliniko API.
type Communication struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// PaginatedLinks contains pagination links.
type PaginatedLinks struct {
	Self string  `json:"self"`
	Next *string `json:"next,omitempty"`
}

// PatientsResponse represents a paginated response of patients.
type PatientsResponse struct {
	Patients []Patient      `json:"patients"`
	Links    PaginatedLinks `json:"links"`
}

// AppointmentsResponse represents a paginated response of appointments.
type AppointmentsResponse struct {
	Appointments []Appointment  `json:"appointments"`
	Links        PaginatedLinks `json:"links"`
}

// TreatmentNotesResponse represents a paginated response of treatment notes.
type TreatmentNotesResponse struct {
	TreatmentNotes []TreatmentNote `json:"treatment_notes"`
	Links          PaginatedLinks  `json:"links"`
}

// CommunicationsResponse represents a paginated response of communications.
type CommunicationsResponse struct {
	Communications []Communication `json:"communications"`
	Links          PaginatedLinks  `json:"links"`
}
