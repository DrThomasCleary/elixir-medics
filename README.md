# Elixir Medics - Cliniko NHS Invoice Extractor

## What's this for?

This tool extracts patient information from Cliniko that needs to be sent to the NHS for purposes of invoicing. It outputs CSV files with patient appointment data.

## Download (Windows)

1. Go to the [Releases page](https://github.com/DrThomasCleary/elixir-medics/releases/latest)
2. Download **elixir-medics-windows.zip**
3. Extract the zip file
4. Double-click **elixir-medics.exe** to launch the app

> **First-time setup:** Windows may show a "Windows protected your PC" warning. Click **More info** then **Run anyway**. This only happens once. After the app opens, click **API Key Settings** at the bottom to enter your Cliniko API key.

## Usage

### GUI Mode (default)

```bash
# Just run the app - GUI opens automatically
./elixir-medics

# The GUI provides:
# - API key settings (stored securely in system keychain)
# - Month/Year selection dropdowns
# - Generate button with progress bar
# - Save buttons for each CSV file
```

### CLI Mode

```bash
# Set API key in .env file or environment
export CLINIKO_API_KEY=your_api_key

# Generate report for a specific month (auto-detects CLI mode)
./elixir-medics --month 11 --year 2024

# Specify output directory
./elixir-medics --month 11 --year 2024 --output-dir ./reports

# Or use --cli flag explicitly
./elixir-medics --cli
```

## Requirements

- Go 1.21+ for building
- A `.env` file with `CLINIKO_API_KEY` set (or set in environment)

## Output Format

### invoice.csv

The invoice CSV only includes **initial assessments** (follow-up appointments are excluded). It contains the following columns:

| Column | Description |
|--------|-------------|
| Reference Number | Formatted as `EML-C382005-2025-XXX` (3-digit zero-padded) |
| Date of Referral | From `custom_fields` (Referral date), or "N/A" if missing |
| Referring GP | Extracted from `patient.notes` (see GP Parsing below) |
| Date of Assessment | British format with London timezone (e.g., "18/10/2025, 13:00") |
| Type | "Initial" (follow-ups are excluded from invoice) |
| Mode | "Face-to-face" or "Remote" |
| Medication | "Yes" or "No" |
| Cost | In pounds |

### patients.csv

A summary of all patients (not filtered by date/time) with the following columns:

| Column | Description |
|--------|-------------|
| Reference Number | Formatted as `EML-C382005-2025-XXX` (3-digit zero-padded) |
| Date of Referral | From `custom_fields` (Referral date), or "N/A" if missing |
| Referring GP | Extracted from `patient.notes` |
| Mode | "Face-to-face" or "Remote" (determined by initial appointment) |
| Medication | "Yes", "No", or "N/A" |
| Discharge Date | From `custom_fields`, or "N/A" if missing |
| Positive Diagnosis | "Yes", "No", or "N/A" |
| Yearly Follow Up | Selected option value, or "N/A" if missing |
| Previous Diagnosis | "Yes", "No", or "N/A" |
| Shared Care | "Yes", "No", or "N/A" |
| Number of Appointments | Total count of appointments for this patient |
| Number of Treatment Notes | Total count of treatment notes for this patient |
| Number of Communications | Total count of communications for this patient |

Patients are sorted by reference number.

### submissions.csv

When a month/year filter is specified, an additional submissions report is generated with summary statistics:

| Field | Description |
|-------|-------------|
| Number of patients on caseload | Patients referred but not discharged by end of month |
| Number of DNA contacts | Appointments where patients did not arrive |
| Percentage DNA | DNA as percentage of total appointments |
| Number of patients who received initial assessment | Patients with first arrived appointment that month |
| Number of patients receiving psychological therapies | Patients with initial assessment + positive diagnosis |
| Number of patients diagnosed with ADHD with new diagnosis | Positive diagnosis without previous diagnosis |
| Percentage of patients diagnosed with ADHD of new diagnosis referrals | Percentage of new diagnoses |
| Number of patients prescribed medication under Shared Care | Patients on caseload with medication = "Yes" AND shared care = "Yes" |
| Number of Referrals | Patients whose referral date is in that month |
| Referrals with previous diagnosis | Referrals where patient has previous diagnosis = "Yes" |
| Referrals without previous diagnosis | Referrals where patient has previous diagnosis = "No" or "N/A" |
| Number of patient contacts | Treatment notes + communications created that month |

## Custom Fields

Patient data is extracted from Cliniko's `custom_fields` property. Fields are matched by **token** (not name, as display names can change).

### Structure

```
custom_fields: {
  sections: [{
    name: "Elixir",
    token: "...",
    fields: [
      { name: "...", type: "date", token: "...", value: "2025-01-01" },
      { name: "...", type: "checkboxes", token: "...", options: [
        { name: "...", token: "...", selected: true }
      ]}
    ]
  }]
}
```

### Field Tokens

| Field | Token | Type |
|-------|-------|------|
| Referral date | `4d1cf1ed-d346-468a-8e0d-23e9525e866b` | date |
| Discharged date | `2b8a8cce-4a6f-4ab1-8deb-4dd8cac7cd7b` | date |
| Diagnosed | `720db7f7-a3bf-45c4-833c-fea5bac6cdab` | checkboxes |
| Diagnosed → Positive | `7eceed68-fff9-4dc4-a06b-8042ba690953` | option |
| Medication | `d152c6e3-b2ce-44ff-843c-3ef0825519c6` | checkboxes |
| Medication → Prescribed | `4f7da0a8-a112-40aa-8727-63046c788799` | option |
| Yearly follow up | `9466f420-4f30-422b-a750-159739730fc7` | checkboxes |
| Previous diagnosis | `330f7335-b6a2-4fb1-bbfe-81c31c0f0936` | checkboxes |
| Previous diagnosis → Yes | `979f3616-9777-4feb-b149-4eca93a85ff9` | option |
| Shared care | `01e24511-3203-4451-b72e-b5428794c39d` | radiobuttons |
| Shared care → Yes | `3010f410-76e5-4d53-9812-16a5fe0bf266` | option |

### Missing Data

If `custom_fields` is null or a field is missing, the value is "N/A". There is no fallback to other patient fields.

## Data Extraction Rules

### Patient Selection

Only patients with `old_reference_id` set are included. For each patient, only the initial assessment is included in the invoice CSV (follow-up appointments are tracked internally but excluded from the invoice output).

### Reference Number Formatting

The reference number in Cliniko (e.g., `EML12`, `EML 8`, or even `ELM13` with typo) is transformed to the NHS format:

- `EML1` → `EML-C382005-2025-001`
- `EML 13` → `EML-C382005-2025-013`
- `ELM5` (typo) → `EML-C382005-2025-005`

The NHS ID `C382005-2025` is always included. The number is zero-padded to 3 digits. Any `ELM` typos are corrected to `EML`.

### GP Parsing

The referring GP information is extracted from `patient.notes`:

1. Find a line that **starts with** `GP` (case insensitive) - this line may contain other text like "GP Surgery" or "GP summary"
2. **Ignore that line entirely**
3. Capture **all lines after** the GP line (including multiple lines with address, email, etc.)

Example notes:
```
GP Surgery
Dr Smith
123 High Street
London
SW1 1AA
doctor@nhs.net
```

Result: "Dr Smith\n123 High Street\nLondon\nSW1 1AA\ndoctor@nhs.net"

### Appointment Selection

1. Fetch all appointments for the patient
2. Sort by date ascending
3. Filter to appointments where `did_not_arrive` is `false`
4. The first valid appointment = Initial assessment
5. The second valid appointment = Follow-up
6. All subsequent appointments are ignored

### Mode (Face-to-face vs Remote)

The mode is determined **only by the initial appointment**:

- Check if `appointment_type.name` contains "Remote" (case insensitive)
- If the initial is Remote, **both rows** (Initial and Follow-up) are marked as Remote
- If the initial is Face-to-face, **both rows** are marked as Face-to-face

The follow-up appointment's actual type is irrelevant.

### Medication

Medication status is extracted from the `custom_fields` Medication checkbox field. If "Prescribed" is selected, the value is "Yes". Otherwise "No". If `custom_fields` is null or missing, the value is "N/A".

### Cost

The cost is only charged for the initial appointment, with the following rules:

- if it's face-to-face, it's 1025 pounds
- if it's online, it's 925 pounds
- if there is medication, there's an additional 400 pounds

### Date Formatting

All appointment dates are converted to:
- **London timezone** (handles BST/GMT automatically)
- **British format**: `DD/MM/YYYY, HH:MM`

### Output Filtering and Sorting

1. **Future appointments are excluded** - only appointments up to the current date/time are included
2. **Sorted by appointment date ascending** - earliest appointments first

## Examples

### Example 1: Two appointments, both attended

Patient appointments:
- 10/10/2025 (showed up)
- 15/10/2025 (showed up)

Invoice output: One row (only initial assessment)
- Initial assessment on 10/10/2025, Medication: Yes

### Example 2: Three appointments

Patient appointments:
- 24/09/2025 (showed up)
- 12/10/2025 (showed up)
- 19/10/2025 (showed up)

Invoice output: One row (only initial assessment)
- Initial assessment on 24/09/2025

### Example 3: One attended, one missed

Patient appointments:
- 08/11/2025 (showed up)
- 06/12/2025 (did not show up)

Invoice output: One row
- Initial assessment on 08/11/2025, Medication: Yes

### Example 4: Remote initial

Patient appointments:
- 10/10/2025 - Remote appointment (showed up)
- 15/10/2025 - Face-to-face appointment (showed up)

Invoice output: One row
- Initial on 10/10/2025, Mode: Remote

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                           main.go                               │
│         Parses flags, reads env, orchestrates report            │
└─────────────────────────────────┬───────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Report Generator (internal/report/)          │
│  - Coordinates data fetching and processing                     │
│  - Worker pool (5 concurrent) for patient processing            │
│  - Generates submissions statistics                             │
│  - Filters by date/month/year                                   │
└────────────┬─────────────────┬─────────────────┬────────────────┘
             │                 │                 │
             ▼                 ▼                 ▼
┌────────────────────┐ ┌─────────────────┐ ┌──────────────────────┐
│ Cliniko Client     │ │ Data Extractor  │ │ CSV Writer           │
│ (internal/cliniko/)│ │ (internal/      │ │ (internal/csv/)      │
│                    │ │  extract/)      │ │                      │
│ - Rate limiting    │ │ - Custom fields │ │ - Invoice CSV        │
│   (200 req/min)    │ │ - Ref number    │ │ - Appointments CSV   │
│ - 429 retry with   │ │   formatting    │ │ - Patients CSV       │
│   backoff          │ │ - GP parsing    │ │ - Submissions CSV    │
│ - Pagination       │ │ - Cost calc     │ │ - Proper escaping    │
│ - Basic Auth       │ │ - Date format   │ │                      │
└────────────────────┘ └─────────────────┘ └──────────────────────┘
```

## Project Structure

```
elixir-medics/
├── main.go                     # Entry point (GUI or CLI)
├── go.mod
├── go.sum
└── internal/
    ├── cliniko/
    │   ├── client.go           # Client interface
    │   ├── client_impl.go      # Implementation with rate limiting
    │   ├── client_test.go      # Tests
    │   ├── models.go           # API response structs
    │   └── mock.go             # Mock client for testing
    ├── extract/
    │   ├── types.go            # Data types
    │   ├── extract.go          # Main extraction logic
    │   ├── customfields.go     # Custom field parsing
    │   ├── gp.go               # GP parsing from notes
    │   └── extract_test.go     # Tests
    ├── report/
    │   ├── types.go            # Report types
    │   ├── generator.go        # Report orchestration
    │   ├── submissions.go      # Submissions report
    │   └── generator_test.go   # Tests
    ├── csv/
    │   ├── types.go            # Writer interface
    │   ├── writer.go           # CSV generation
    │   └── writer_test.go      # Tests
    ├── ui/
    │   └── app.go              # Fyne GUI application
    └── keychain/
        └── keychain.go         # Secure API key storage
```

## Technical Details

### Rate Limiting

The Cliniko API has a limit of 200 requests per minute. We use a token bucket rate limiter:

```go
limiter := rate.NewLimiter(rate.Limit(200.0/60.0), 1)
```

### 429 Retry Handling

HTTP 429 responses are automatically retried with exponential backoff using `go-retryablehttp`.

### API Endpoints Used

- `GET /patients` - Paginated list of all patients
- `GET /patients/{id}/appointments` - Appointments for a specific patient
- `GET /appointment_types/{id}` - Appointment type details (to check for "Remote")
- `GET /treatment_notes?q[]=patient_id:={id}` - Treatment notes for a specific patient
- `GET /communications?q[]=patient_id:={id}` - Communications for a specific patient

### Concurrency

Patient processing uses a worker pool of 5 concurrent goroutines to avoid overwhelming the Cliniko API.

## Building

```bash
# Build for current platform
make build

# Build Windows executable (requires fyne-cross and Docker)
make build-windows
```

### Windows Build Prerequisites

Building for Windows requires [fyne-cross](https://github.com/fyne-io/fyne-cross) and Docker:

```bash
go install github.com/fyne-io/fyne-cross@latest
```

The Windows executable will be output to `fyne-cross/dist/windows-amd64/elixir-medics.exe.zip`.

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Verbose output
go test -v ./...
```
