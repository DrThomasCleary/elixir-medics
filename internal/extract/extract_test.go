package extract

import (
	"context"
	"testing"

	"github.com/schani/elixir-medics/internal/cliniko"
)

// TestFormatReferenceNumber tests reference number formatting.
func TestFormatReferenceNumber(t *testing.T) {
	tests := []struct {
		name     string
		refID    string
		expected string
	}{
		{
			name:     "single digit",
			refID:    "EML1",
			expected: "EML-C382005-2025-001",
		},
		{
			name:     "double digit",
			refID:    "EML12",
			expected: "EML-C382005-2025-012",
		},
		{
			name:     "triple digit",
			refID:    "EML123",
			expected: "EML-C382005-2025-123",
		},
		{
			name:     "with space",
			refID:    "EML 13",
			expected: "EML-C382005-2025-013",
		},
		{
			name:     "no number",
			refID:    "EML",
			expected: "EML-C382005-2025-000",
		},
		{
			name:     "empty string",
			refID:    "",
			expected: "EML-C382005-2025-000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatReferenceNumber(tt.refID)
			if result != tt.expected {
				t.Errorf("FormatReferenceNumber(%q) = %q, want %q", tt.refID, result, tt.expected)
			}
		})
	}
}

// TestFormatDateBritish tests date formatting.
func TestFormatDateBritish(t *testing.T) {
	tests := []struct {
		name     string
		isoDate  string
		expected string
	}{
		{
			name:     "valid UTC date",
			isoDate:  "2025-10-15T10:00:00Z",
			expected: "15/10/2025, 11:00", // BST +1
		},
		{
			name:     "valid date with timezone",
			isoDate:  "2025-01-20T14:30:00Z",
			expected: "20/01/2025, 14:30", // GMT +0
		},
		{
			name:     "invalid date",
			isoDate:  "invalid",
			expected: "N/A",
		},
		{
			name:     "empty string",
			isoDate:  "",
			expected: "N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDateBritish(tt.isoDate)
			if result != tt.expected {
				t.Errorf("FormatDateBritish(%q) = %q, want %q", tt.isoDate, result, tt.expected)
			}
		})
	}
}

// TestCalculateCost tests cost calculation logic.
func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name          string
		isRemote      bool
		hasMedication bool
		expectedCost  int
	}{
		{
			name:          "face-to-face without medication",
			isRemote:      false,
			hasMedication: false,
			expectedCost:  1025,
		},
		{
			name:          "face-to-face with medication",
			isRemote:      false,
			hasMedication: true,
			expectedCost:  1425,
		},
		{
			name:          "remote without medication",
			isRemote:      true,
			hasMedication: false,
			expectedCost:  925,
		},
		{
			name:          "remote with medication",
			isRemote:      true,
			hasMedication: true,
			expectedCost:  1325,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCost(tt.isRemote, tt.hasMedication)
			if result != tt.expectedCost {
				t.Errorf("CalculateCost(%v, %v) = %d, want %d", tt.isRemote, tt.hasMedication, result, tt.expectedCost)
			}
		})
	}
}

// TestParseGP tests GP extraction from patient notes.
func TestParseGP(t *testing.T) {
	tests := []struct {
		name     string
		notes    *string
		expected string
	}{
		{
			name:     "nil notes",
			notes:    nil,
			expected: "N/A",
		},
		{
			name:     "empty notes",
			notes:    strPtr(""),
			expected: "N/A",
		},
		{
			name:     "GP with multiple lines",
			notes:    strPtr("GP\nDr Smith\n123 High Street\nLondon"),
			expected: "Dr Smith\n123 High Street\nLondon",
		},
		{
			name:     "GP lowercase",
			notes:    strPtr("gp\nDr Jones"),
			expected: "Dr Jones",
		},
		{
			name:     "GP uppercase",
			notes:    strPtr("GP\nMedical Centre"),
			expected: "Medical Centre",
		},
		{
			name:     "no GP line",
			notes:    strPtr("Some notes\nAbout patient"),
			expected: "N/A",
		},
		{
			name:     "GP with whitespace",
			notes:    strPtr("  GP  \nDr Brown"),
			expected: "Dr Brown",
		},
		{
			name:     "GP line only, no following lines",
			notes:    strPtr("GP"),
			expected: "N/A",
		},
		{
			name:     "GP Surgery format",
			notes:    strPtr("GP Surgery\nDr Smith\n123 High Street\nLondon"),
			expected: "Dr Smith\n123 High Street\nLondon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGP(tt.notes)
			if result != tt.expected {
				t.Errorf("ParseGP() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestParseCustomFields tests custom field extraction.
func TestParseCustomFields(t *testing.T) {
	tests := []struct {
		name     string
		patient  cliniko.Patient
		expected ParsedCustomFields
	}{
		{
			name: "nil custom fields",
			patient: cliniko.Patient{
				CustomFields: nil,
			},
			expected: ParsedCustomFields{Medication: MedStatusNA},
		},
		{
			name: "empty custom fields",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{},
				},
			},
			expected: ParsedCustomFields{Medication: MedStatusNA},
		},
		{
			name: "referral date only",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenReferralDate,
									Value: strPtr("2025-09-01"),
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				ReferralDate: strPtr("2025-09-01"),
				Medication:   MedStatusNA,
			},
		},
		{
			name: "medication prescribed",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenMedication,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenMedicationPrescribed,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication: MedStatusYes,
			},
		},
		{
			name: "medication not prescribed",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenMedication,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenMedicationPrescribed,
											Selected: false,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication: MedStatusNo,
			},
		},
		{
			name: "positive diagnosis",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenDiagnosed,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenDiagnosedPositive,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication:        MedStatusNA,
				PositiveDiagnosis: boolPtr(true),
			},
		},
		{
			name: "previous diagnosis yes",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenPreviousDiagnosis,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenPreviousDiagnosisYes,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication:        MedStatusNA,
				PreviousDiagnosis: boolPtr(true),
			},
		},
		{
			name: "shared care yes",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenSharedCare,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenSharedCareYes,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication: MedStatusNA,
				SharedCare: boolPtr(true),
			},
		},
		{
			name: "shared care no",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenSharedCare,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    "other-token",
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication: MedStatusNA,
				SharedCare: boolPtr(false),
			},
		},
		{
			name: "yearly follow up",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenYearlyFollowUp,
									Options: []cliniko.CustomFieldOption{
										{
											Name:     "2026",
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				Medication:     MedStatusNA,
				YearlyFollowUp: strPtr("2026"),
			},
		},
		{
			name: "all fields populated",
			patient: cliniko.Patient{
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenReferralDate,
									Value: strPtr("2025-09-01"),
								},
								{
									Token: TokenDischargeDate,
									Value: strPtr("2025-12-01"),
								},
								{
									Token: TokenMedication,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenMedicationPrescribed,
											Selected: true,
										},
									},
								},
								{
									Token: TokenDiagnosed,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenDiagnosedPositive,
											Selected: true,
										},
									},
								},
								{
									Token: TokenYearlyFollowUp,
									Options: []cliniko.CustomFieldOption{
										{
											Name:     "2026",
											Selected: true,
										},
									},
								},
								{
									Token: TokenPreviousDiagnosis,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenPreviousDiagnosisYes,
											Selected: true,
										},
									},
								},
								{
									Token: TokenSharedCare,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenSharedCareYes,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: ParsedCustomFields{
				ReferralDate:      strPtr("2025-09-01"),
				DischargeDate:     strPtr("2025-12-01"),
				Medication:        MedStatusYes,
				PositiveDiagnosis: boolPtr(true),
				YearlyFollowUp:    strPtr("2026"),
				PreviousDiagnosis: boolPtr(true),
				SharedCare:        boolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCustomFields(tt.patient)
			compareCustomFields(t, result, tt.expected)
		})
	}
}

// TestExtractPatientRows tests the main extraction logic.
func TestExtractPatientRows(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		patient          cliniko.Patient
		appointments     []cliniko.Appointment
		appointmentTypes map[string]*cliniko.AppointmentType
		expectedRows     int
		expectedType     AppointmentType
		expectedMode     AppointmentMode
		expectedCost     string
	}{
		{
			name: "patient without reference ID",
			patient: cliniko.Patient{
				ID:             1,
				FirstName:      "No",
				LastName:       "Ref",
				OldReferenceID: nil,
			},
			appointments:     []cliniko.Appointment{},
			appointmentTypes: map[string]*cliniko.AppointmentType{},
			expectedRows:     0,
		},
		{
			name: "patient with empty reference ID",
			patient: cliniko.Patient{
				ID:             1,
				FirstName:      "No",
				LastName:       "Ref",
				OldReferenceID: strPtr(""),
			},
			appointments:     []cliniko.Appointment{},
			appointmentTypes: map[string]*cliniko.AppointmentType{},
			expectedRows:     0,
		},
		{
			name: "patient with no appointments",
			patient: cliniko.Patient{
				ID:             1,
				FirstName:      "Jane",
				LastName:       "Doe",
				OldReferenceID: strPtr("EML1"),
			},
			appointments:     []cliniko.Appointment{},
			appointmentTypes: map[string]*cliniko.AppointmentType{},
			expectedRows:     1,
			expectedType:     TypeNA,
			expectedMode:     ModeNA,
			expectedCost:     "",
		},
		{
			name: "patient with one face-to-face appointment",
			patient: cliniko.Patient{
				ID:             1,
				FirstName:      "Jane",
				LastName:       "Smith",
				OldReferenceID: strPtr("EML1"),
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt1",
					AppointmentStart: "2025-10-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/1": {
					ID:   1,
					Name: "Face-to-face Consultation",
				},
			},
			expectedRows: 1,
			expectedType: TypeInitial,
			expectedMode: ModeFaceToFace,
			expectedCost: "£1025",
		},
		{
			name: "patient with remote appointment",
			patient: cliniko.Patient{
				ID:             2,
				FirstName:      "Bob",
				LastName:       "Jones",
				OldReferenceID: strPtr("EML2"),
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt2",
					AppointmentStart: "2025-10-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/2",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/2": {
					ID:   2,
					Name: "Remote Consultation",
				},
			},
			expectedRows: 1,
			expectedType: TypeInitial,
			expectedMode: ModeRemote,
			expectedCost: "£925",
		},
		{
			name: "patient with medication",
			patient: cliniko.Patient{
				ID:             3,
				FirstName:      "Alice",
				LastName:       "Brown",
				OldReferenceID: strPtr("EML3"),
				CustomFields: &cliniko.CustomFields{
					Sections: []cliniko.CustomFieldSection{
						{
							Fields: []cliniko.CustomField{
								{
									Token: TokenMedication,
									Options: []cliniko.CustomFieldOption{
										{
											Token:    TokenMedicationPrescribed,
											Selected: true,
										},
									},
								},
							},
						},
					},
				},
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt3",
					AppointmentStart: "2025-10-15T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/1": {
					ID:   1,
					Name: "Face-to-face Consultation",
				},
			},
			expectedRows: 1,
			expectedType: TypeInitial,
			expectedMode: ModeFaceToFace,
			expectedCost: "£1425", // 1025 + 400
		},
		{
			name: "patient with initial and follow-up",
			patient: cliniko.Patient{
				ID:             4,
				FirstName:      "Charlie",
				LastName:       "Davis",
				OldReferenceID: strPtr("EML4"),
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt4",
					AppointmentStart: "2025-10-10T09:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
				{
					ID:               "apt5",
					AppointmentStart: "2025-10-20T14:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/1": {
					ID:   1,
					Name: "Face-to-face Consultation",
				},
			},
			expectedRows: 2,
			expectedType: TypeInitial,
			expectedMode: ModeFaceToFace,
			expectedCost: "£1025",
		},
		{
			name: "patient with DNA appointment",
			patient: cliniko.Patient{
				ID:             5,
				FirstName:      "Dave",
				LastName:       "Evans",
				OldReferenceID: strPtr("EML5"),
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt6",
					AppointmentStart: "2025-10-10T09:00:00Z",
					DidNotArrive:     true, // DNA
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
				{
					ID:               "apt7",
					AppointmentStart: "2025-10-20T14:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/1": {
					ID:   1,
					Name: "Face-to-face Consultation",
				},
			},
			expectedRows: 1,
			expectedType: TypeInitial, // Second appointment becomes initial
			expectedMode: ModeFaceToFace,
			expectedCost: "£1025",
		},
		{
			name: "patient with more than 2 appointments",
			patient: cliniko.Patient{
				ID:             6,
				FirstName:      "Eve",
				LastName:       "Foster",
				OldReferenceID: strPtr("EML6"),
			},
			appointments: []cliniko.Appointment{
				{
					ID:               "apt8",
					AppointmentStart: "2025-10-10T09:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
				{
					ID:               "apt9",
					AppointmentStart: "2025-10-20T14:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
				{
					ID:               "apt10",
					AppointmentStart: "2025-11-05T10:00:00Z",
					DidNotArrive:     false,
					AppointmentType: cliniko.AppointmentTypeLink{
						Links: struct {
							Self string `json:"self"`
						}{
							Self: "https://api.cliniko.com/v1/appointment_types/1",
						},
					},
				},
			},
			appointmentTypes: map[string]*cliniko.AppointmentType{
				"https://api.cliniko.com/v1/appointment_types/1": {
					ID:   1,
					Name: "Face-to-face Consultation",
				},
			},
			expectedRows: 2, // Only first 2
			expectedType: TypeInitial,
			expectedMode: ModeFaceToFace,
			expectedCost: "£1025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{
				appointments:     tt.appointments,
				appointmentTypes: tt.appointmentTypes,
			}

			result, err := ExtractPatientRows(ctx, client, tt.patient)
			if err != nil {
				t.Fatalf("ExtractPatientRows() error = %v", err)
			}

			if len(result.Rows) != tt.expectedRows {
				t.Errorf("got %d rows, want %d", len(result.Rows), tt.expectedRows)
			}

			if tt.expectedRows > 0 {
				firstRow := result.Rows[0]
				if firstRow.Type != tt.expectedType {
					t.Errorf("got type %q, want %q", firstRow.Type, tt.expectedType)
				}
				if firstRow.Mode != tt.expectedMode {
					t.Errorf("got mode %q, want %q", firstRow.Mode, tt.expectedMode)
				}
				if firstRow.Cost != tt.expectedCost {
					t.Errorf("got cost %q, want %q", firstRow.Cost, tt.expectedCost)
				}
			}

			// Verify follow-up has no cost
			if tt.expectedRows == 2 {
				if result.Rows[1].Type != TypeFollowUp {
					t.Errorf("second row type = %q, want %q", result.Rows[1].Type, TypeFollowUp)
				}
				if result.Rows[1].Cost != "" {
					t.Errorf("follow-up cost = %q, want empty", result.Rows[1].Cost)
				}
				// Follow-up should have same mode as initial
				if result.Rows[1].Mode != result.Rows[0].Mode {
					t.Errorf("follow-up mode %q != initial mode %q", result.Rows[1].Mode, result.Rows[0].Mode)
				}
			}
		})
	}
}

// Mock client for testing
type mockClient struct {
	appointments     []cliniko.Appointment
	appointmentTypes map[string]*cliniko.AppointmentType
}

func (m *mockClient) GetAllPatients(ctx context.Context) (<-chan cliniko.Patient, <-chan error) {
	patients := make(chan cliniko.Patient)
	errors := make(chan error)
	close(patients)
	close(errors)
	return patients, errors
}

func (m *mockClient) GetAppointmentsForPatient(ctx context.Context, patientID int64) ([]cliniko.Appointment, error) {
	return m.appointments, nil
}

func (m *mockClient) GetAppointmentType(ctx context.Context, url string) (*cliniko.AppointmentType, error) {
	if apt, ok := m.appointmentTypes[url]; ok {
		return apt, nil
	}
	return nil, nil
}

func (m *mockClient) GetTreatmentNotesForPatient(ctx context.Context, patientID int64) ([]cliniko.TreatmentNote, error) {
	return []cliniko.TreatmentNote{}, nil
}

func (m *mockClient) GetCommunicationsForPatient(ctx context.Context, patientID int64) ([]cliniko.Communication, error) {
	return []cliniko.Communication{}, nil
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func compareCustomFields(t *testing.T, got, want ParsedCustomFields) {
	t.Helper()

	if !comparePtrString(got.ReferralDate, want.ReferralDate) {
		t.Errorf("ReferralDate = %v, want %v", ptrToString(got.ReferralDate), ptrToString(want.ReferralDate))
	}
	if !comparePtrString(got.DischargeDate, want.DischargeDate) {
		t.Errorf("DischargeDate = %v, want %v", ptrToString(got.DischargeDate), ptrToString(want.DischargeDate))
	}
	if got.Medication != want.Medication {
		t.Errorf("Medication = %v, want %v", got.Medication, want.Medication)
	}
	if !comparePtrBool(got.PositiveDiagnosis, want.PositiveDiagnosis) {
		t.Errorf("PositiveDiagnosis = %v, want %v", ptrToBool(got.PositiveDiagnosis), ptrToBool(want.PositiveDiagnosis))
	}
	if !comparePtrString(got.YearlyFollowUp, want.YearlyFollowUp) {
		t.Errorf("YearlyFollowUp = %v, want %v", ptrToString(got.YearlyFollowUp), ptrToString(want.YearlyFollowUp))
	}
	if !comparePtrBool(got.PreviousDiagnosis, want.PreviousDiagnosis) {
		t.Errorf("PreviousDiagnosis = %v, want %v", ptrToBool(got.PreviousDiagnosis), ptrToBool(want.PreviousDiagnosis))
	}
	if !comparePtrBool(got.SharedCare, want.SharedCare) {
		t.Errorf("SharedCare = %v, want %v", ptrToBool(got.SharedCare), ptrToBool(want.SharedCare))
	}
}

func comparePtrString(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func comparePtrBool(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrToString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func ptrToBool(b *bool) string {
	if b == nil {
		return "<nil>"
	}
	if *b {
		return "true"
	}
	return "false"
}
