package report

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/schani/elixir-medics/internal/csv"
	"github.com/schani/elixir-medics/internal/debuglog"
	"github.com/schani/elixir-medics/internal/extract"
)

const concurrencyLimit = 5

// Generator generates reports from Cliniko data.
type Generator struct {
	client    cliniko.Client
	csvWriter csv.Writer
}

// NewGenerator creates a new report generator.
func NewGenerator(client cliniko.Client) *Generator {
	return &Generator{
		client:    client,
		csvWriter: csv.NewCSVWriter(),
	}
}

// Generate generates all reports.
func (g *Generator) Generate(ctx context.Context, opts Options) (*Result, error) {
	log.Println("[DEBUG] Generate: starting")
	runID := "no-month"
	if opts.Month != nil && opts.Year != nil {
		runID = fmt.Sprintf("%04d-%02d", *opts.Year, *opts.Month)
	}
	now := time.Now()
	if opts.Now != nil {
		now = *opts.Now
	}

	// Collect all patients first
	log.Println("[DEBUG] Generate: collecting patients...")
	patients, err := g.collectPatients(ctx, opts)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Generate: collected %d patients", len(patients))

	// Process patients concurrently with limit
	log.Println("[DEBUG] Generate: processing patients...")
	allRows, allAppointments, err := g.processPatients(ctx, patients, opts)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Generate: processed, got %d rows, %d appointments", len(allRows), len(allAppointments))

	// Filter out future appointments and rows with no date
	var filteredRows []extract.ExtractedRow
	for _, row := range allRows {
		if row.DateOfAssessmentRaw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, row.DateOfAssessmentRaw)
		if err != nil {
			continue
		}
		if t.After(now) {
			continue
		}
		filteredRows = append(filteredRows, row)
	}

	// Build lookup: does this patient have a titration in ANY past month?
	// (before month filtering so we can see the full appointment history)
	hasTitrationAnyMonth := make(map[string]bool)
	for _, row := range filteredRows {
		if row.Type == extract.TypeTitration {
			hasTitrationAnyMonth[row.ReferenceNumber] = true
		}
	}

	// Filter by month/year if specified
	if opts.Month != nil && opts.Year != nil {
		var monthFiltered []extract.ExtractedRow
		for _, row := range filteredRows {
			t, err := time.Parse(time.RFC3339, row.DateOfAssessmentRaw)
			if err != nil {
				continue
			}
			if int(t.Month()) == *opts.Month && t.Year() == *opts.Year {
				monthFiltered = append(monthFiltered, row)
			}
		}
		filteredRows = monthFiltered
	}

	// Sort by appointment date ascending
	sort.Slice(filteredRows, func(i, j int) bool {
		t1, _ := time.Parse(time.RFC3339, filteredRows[i].DateOfAssessmentRaw)
		t2, _ := time.Parse(time.RFC3339, filteredRows[j].DateOfAssessmentRaw)
		return t1.Before(t2)
	})

	// Invoice rules:
	// - Initial: include always. Add £400 surcharge only if patient has medication
	//   AND no titration appointment exists in any month (medication done during assessment).
	// - Titration: include in whichever month it falls in, charged at £400.
	// - Follow-up: excluded (no charge).
	var invoiceRows []extract.ExtractedRow
	for _, row := range filteredRows {
		if row.Type == extract.TypeInitial {
			if row.Medication.HasMedication() && !hasTitrationAnyMonth[row.ReferenceNumber] {
				row.Cost = fmt.Sprintf("£%d", extract.CalculateInitialCost(row.Mode == extract.ModeRemote)+extract.TitrationCost)
			}
			invoiceRows = append(invoiceRows, row)
		} else if row.Type == extract.TypeTitration {
			invoiceRows = append(invoiceRows, row)
		}
	}
	if opts.Month != nil && opts.Year != nil && *opts.Month == 3 && *opts.Year == 2026 {
		extractedInitial := 0
		extractedTitration := 0
		extractedFollowUp := 0
		for _, row := range filteredRows {
			switch row.Type {
			case extract.TypeInitial:
				extractedInitial++
			case extract.TypeTitration:
				extractedTitration++
			case extract.TypeFollowUp:
				extractedFollowUp++
			}
		}
		invoiceInitial := 0
		invoiceTitration := 0
		for _, row := range invoiceRows {
			switch row.Type {
			case extract.TypeInitial:
				invoiceInitial++
			case extract.TypeTitration:
				invoiceTitration++
			}
		}
		// #region agent log
		debuglog.Log(runID, "H2", "internal/report/generator.go:134", "march row summary before and after invoice filtering", map[string]any{
			"filteredRows":        len(filteredRows),
			"extractedInitial":    extractedInitial,
			"extractedTitration":  extractedTitration,
			"extractedFollowUp":   extractedFollowUp,
			"invoiceRows":         len(invoiceRows),
			"invoiceInitial":      invoiceInitial,
			"invoiceTitration":    invoiceTitration,
		})
		// #endregion
	}

	invoiceCSV := g.csvWriter.WriteInvoice(invoiceRows)

	// Filter appointments for output (exclude future only, no month/year filter)
	var filteredAppointments []extract.AppointmentWithPatient
	for _, apt := range allAppointments {
		t, err := time.Parse(time.RFC3339, apt.AppointmentDateTimeRaw)
		if err != nil {
			continue
		}
		if !t.After(now) {
			filteredAppointments = append(filteredAppointments, apt)
		}
	}

	// Sort appointments by date ascending
	sort.Slice(filteredAppointments, func(i, j int) bool {
		t1, _ := time.Parse(time.RFC3339, filteredAppointments[i].AppointmentDateTimeRaw)
		t2, _ := time.Parse(time.RFC3339, filteredAppointments[j].AppointmentDateTimeRaw)
		return t1.Before(t2)
	})

	appointmentsCSV := g.csvWriter.WriteAppointments(filteredAppointments)

	// Filter appointments by month/year for the monthly sheet
	var monthlyAppointmentsCSV *string
	if opts.Month != nil && opts.Year != nil {
		var monthlyAppointments []extract.AppointmentWithPatient
		for _, apt := range filteredAppointments {
			t, err := time.Parse(time.RFC3339, apt.AppointmentDateTimeRaw)
			if err != nil {
				continue
			}
			if int(t.Month()) == *opts.Month && t.Year() == *opts.Year {
				monthlyAppointments = append(monthlyAppointments, apt)
			}
		}
		csv := g.csvWriter.WriteAppointments(monthlyAppointments)
		monthlyAppointmentsCSV = &csv
	}
	log.Println("[DEBUG] Generate: wrote appointments CSV")

	// Fetch treatment notes and communications for patients with old_reference_id
	var patientsWithRefID []cliniko.Patient
	for _, p := range patients {
		if p.OldReferenceID != nil && *p.OldReferenceID != "" {
			patientsWithRefID = append(patientsWithRefID, p)
		}
	}
	log.Printf("[DEBUG] Generate: fetching contacts for %d patients with ref ID...", len(patientsWithRefID))

	contactsResults, err := g.fetchContacts(ctx, patientsWithRefID, opts)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Generate: fetched contacts, got %d results", len(contactsResults))

	// Flatten contacts into PatientContact[] and track counts per patient
	var allContacts []extract.PatientContact
	treatmentNotesCountByPatientID := make(map[int64]int)
	communicationsCountByPatientID := make(map[int64]int)

	for _, result := range contactsResults {
		treatmentNotesCountByPatientID[result.patientID] = len(result.treatmentNotes)
		communicationsCountByPatientID[result.patientID] = len(result.communications)
		for _, note := range result.treatmentNotes {
			allContacts = append(allContacts, extract.PatientContact{CreatedAt: note.CreatedAt})
		}
		for _, comm := range result.communications {
			allContacts = append(allContacts, extract.PatientContact{CreatedAt: comm.CreatedAt})
		}
	}

	// Generate submissions report if month/year filter is applied
	var submissionsCSV *string
	if opts.Month != nil && opts.Year != nil {
		report := GenerateSubmissionsReport(filteredAppointments, allContacts, allRows, patients, *opts.Month, *opts.Year)
		csv := g.csvWriter.WriteSubmissions(report)
		submissionsCSV = &csv
	}

	// Generate patients CSV (not filtered by date/time)
	patientRows := g.buildPatientRows(allRows, allAppointments, patientsWithRefID, treatmentNotesCountByPatientID, communicationsCountByPatientID)

	// Sort patients by reference number
	sort.Slice(patientRows, func(i, j int) bool {
		return patientRows[i].ReferenceNumber < patientRows[j].ReferenceNumber
	})

	patientsCSV := g.csvWriter.WritePatients(patientRows)

	// Generate yearly follow-up list:
	// Include patients where 9+ months since discharge AND medication is Yes or Other
	yearlyFollowUpRows := g.buildYearlyFollowUpRows(patientRows, now)
	yearlyFollowUpCSV := g.csvWriter.WriteYearlyFollowUp(yearlyFollowUpRows)

	// Generate 10 weeks waiting list: patients with medication status "10 Weeks Waiting"
	var tenWeeksWaitingRows []extract.TenWeeksWaitingRow
	for _, pr := range patientRows {
		if pr.Medication == extract.MedStatus10WeeksWaiting {
			tenWeeksWaitingRows = append(tenWeeksWaitingRows, extract.TenWeeksWaitingRow{
				PatientName:     pr.PatientName,
				ReferenceNumber: pr.ReferenceNumber,
				DateOfReferral:  pr.DateOfReferral,
				ReferringGP:     pr.ReferringGP,
			})
		}
	}
	tenWeeksWaitingCSV := g.csvWriter.WriteTenWeeksWaiting(tenWeeksWaitingRows)

	// Build missing info list from ALL patients (not just those with EML numbers)
	var missingInfoRows []extract.MissingInfoRow
	for _, p := range patients {
		name := fmt.Sprintf("%s %s", p.FirstName, p.LastName)
		emlNumber := ""
		if p.OldReferenceID != nil {
			emlNumber = *p.OldReferenceID
		}
		cf := extract.ParseCustomFields(p)
		referralDate := ""
		if cf.ReferralDate != nil {
			referralDate = *cf.ReferralDate
		}

		missingEML := emlNumber == ""
		missingReferral := referralDate == ""

		if missingEML || missingReferral {
			missing := ""
			if missingEML && missingReferral {
				missing = "EML Number, Referral Date"
			} else if missingEML {
				missing = "EML Number"
			} else {
				missing = "Referral Date"
			}
			missingInfoRows = append(missingInfoRows, extract.MissingInfoRow{
				PatientName:   name,
				PatientID:     p.ID,
				EMLNumber:     emlNumber,
				ReferralDate:  referralDate,
				MissingFields: missing,
			})
		}
	}
	sort.Slice(missingInfoRows, func(i, j int) bool {
		return missingInfoRows[i].PatientName < missingInfoRows[j].PatientName
	})
	missingInfoCSV := g.csvWriter.WriteMissingInfo(missingInfoRows)

	log.Println("[DEBUG] Generate: all done, returning result")

	return &Result{
		InvoiceCSV:             invoiceCSV,
		AppointmentsCSV:        appointmentsCSV,
		MonthlyAppointmentsCSV: monthlyAppointmentsCSV,
		PatientsCSV:            patientsCSV,
		SubmissionsCSV:         submissionsCSV,
		YearlyFollowUpCSV:      yearlyFollowUpCSV,
		TenWeeksWaitingCSV:     tenWeeksWaitingCSV,
		MissingInfoCSV:         missingInfoCSV,
		RawPatients:            patients,
	}, nil
}

func (g *Generator) collectPatients(ctx context.Context, opts Options) ([]cliniko.Patient, error) {
	var patients []cliniko.Patient
	patientsCh, errCh := g.client.GetAllPatients(ctx)

	for patient := range patientsCh {
		patients = append(patients, patient)
		if len(patients)%100 == 0 && opts.OnPatientsFetched != nil {
			opts.OnPatientsFetched(len(patients), nil)
		}
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	if opts.OnPatientsFetched != nil {
		total := len(patients)
		opts.OnPatientsFetched(len(patients), &total)
	}

	return patients, nil
}

type extractionResult struct {
	rows         []extract.ExtractedRow
	appointments []extract.AppointmentWithPatient
	err          error
}

func (g *Generator) processPatients(ctx context.Context, patients []cliniko.Patient, opts Options) ([]extract.ExtractedRow, []extract.AppointmentWithPatient, error) {
	// Use worker pool pattern
	jobs := make(chan cliniko.Patient, len(patients))
	results := make(chan extractionResult, len(patients))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrencyLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for patient := range jobs {
				result, err := extract.ExtractPatientRows(ctx, g.client, patient)
				if err != nil {
					results <- extractionResult{err: err}
					continue
				}
				results <- extractionResult{
					rows:         result.Rows,
					appointments: result.Appointments,
				}
			}
		}()
	}

	// Send jobs
	for _, p := range patients {
		jobs <- p
	}
	close(jobs)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allRows []extract.ExtractedRow
	var allAppointments []extract.AppointmentWithPatient
	processedCount := 0

	for result := range results {
		if result.err != nil {
			return nil, nil, result.err
		}
		allRows = append(allRows, result.rows...)
		allAppointments = append(allAppointments, result.appointments...)

		processedCount++
		if processedCount%5 == 0 && opts.OnPatientsProcessed != nil {
			opts.OnPatientsProcessed(processedCount, len(patients))
		}
	}

	return allRows, allAppointments, nil
}

type contactsResult struct {
	patientID      int64
	treatmentNotes []cliniko.TreatmentNote
	communications []cliniko.Communication
}

func (g *Generator) fetchContacts(ctx context.Context, patients []cliniko.Patient, opts Options) ([]contactsResult, error) {
	jobs := make(chan cliniko.Patient, len(patients))
	results := make(chan contactsResult, len(patients))
	errors := make(chan error, concurrencyLimit)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrencyLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for patient := range jobs {
				notes, err := g.client.GetTreatmentNotesForPatient(ctx, patient.ID)
				if err != nil {
					errors <- err
					return
				}
				comms, err := g.client.GetCommunicationsForPatient(ctx, patient.ID)
				if err != nil {
					errors <- err
					return
				}
				results <- contactsResult{
					patientID:      patient.ID,
					treatmentNotes: notes,
					communications: comms,
				}
			}
		}()
	}

	// Send jobs
	for _, p := range patients {
		jobs <- p
	}
	close(jobs)

	// Wait and collect
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Check for errors first
	select {
	case err := <-errors:
		if err != nil {
			return nil, err
		}
	default:
	}

	var allResults []contactsResult
	processedCount := 0
	for result := range results {
		allResults = append(allResults, result)
		processedCount++
		if processedCount%5 == 0 && opts.OnContactsFetched != nil {
			opts.OnContactsFetched(processedCount, len(patients))
		}
	}

	return allResults, nil
}

var refNumRegex = regexp.MustCompile(`(\d+)$`)
var refIDRegex = regexp.MustCompile(`(\d+)`)

func (g *Generator) buildPatientRows(
	allRows []extract.ExtractedRow,
	allAppointments []extract.AppointmentWithPatient,
	patientsWithRefID []cliniko.Patient,
	treatmentNotesCount, communicationsCount map[int64]int,
) []extract.PatientRow {
	// Group rows by referenceNumber to get one row per patient
	patientRowsMap := make(map[string]extract.ExtractedRow)
	for _, row := range allRows {
		if _, exists := patientRowsMap[row.ReferenceNumber]; !exists {
			patientRowsMap[row.ReferenceNumber] = row
		}
	}

	// Count appointments per patient (using referralId from appointments)
	appointmentCountByReferralID := make(map[string]int)
	for _, apt := range allAppointments {
		appointmentCountByReferralID[apt.ReferralID]++
	}

	// Build lookup maps from patientsWithRefID: ref number -> patient ID and name
	patientIDByRefNumber := make(map[int]int64)
	patientNameByRefNumber := make(map[int]string)
	for _, patient := range patientsWithRefID {
		if patient.OldReferenceID != nil {
			match := refIDRegex.FindStringSubmatch(*patient.OldReferenceID)
			if len(match) > 1 {
				num, _ := strconv.Atoi(match[1])
				patientIDByRefNumber[num] = patient.ID
				patientNameByRefNumber[num] = patient.FirstName + " " + patient.LastName
			}
		}
	}

	// Build referralID lookup from appointments for appointment counts
	referralIDByRefNumber := make(map[int]string)
	for _, apt := range allAppointments {
		refIDMatch := refIDRegex.FindStringSubmatch(apt.ReferralID)
		if len(refIDMatch) > 1 {
			num, _ := strconv.Atoi(refIDMatch[1])
			if _, exists := referralIDByRefNumber[num]; !exists {
				referralIDByRefNumber[num] = apt.ReferralID
			}
		}
	}

	var patientRows []extract.PatientRow
	for _, row := range patientRowsMap {
		refNumMatch := refNumRegex.FindStringSubmatch(row.ReferenceNumber)
		var patientName string
		var appointmentCount int
		if len(refNumMatch) > 1 {
			refNum, _ := strconv.Atoi(refNumMatch[1])
			patientName = patientNameByRefNumber[refNum]
			if rid, ok := referralIDByRefNumber[refNum]; ok {
				appointmentCount = appointmentCountByReferralID[rid]
			}
		}

		// Get contact counts for this patient
		var treatmentNotesCountVal, communicationsCountVal int
		if len(refNumMatch) > 1 {
			refNum, _ := strconv.Atoi(refNumMatch[1])
			if patientID, exists := patientIDByRefNumber[refNum]; exists {
				treatmentNotesCountVal = treatmentNotesCount[patientID]
				communicationsCountVal = communicationsCount[patientID]
			}
		}

		patientRows = append(patientRows, extract.PatientRow{
			PatientName:            patientName,
			ReferenceNumber:        row.ReferenceNumber,
			DateOfReferral:         row.DateOfReferral,
			ReferringGP:            row.ReferringGP,
			Mode:                   row.Mode,
			Medication:             row.Medication,
			DischargeDate:          row.DischargeDate,
			PositiveDiagnosis:      row.PositiveDiagnosis,
			YearlyFollowUp:         row.YearlyFollowUp,
			PreviousDiagnosis:      row.PreviousDiagnosis,
			SharedCare:             row.SharedCare,
			NumberOfAppointments:   appointmentCount,
			NumberOfTreatmentNotes: treatmentNotesCountVal,
			NumberOfCommunications: communicationsCountVal,
		})
	}

	return patientRows
}

func (g *Generator) buildYearlyFollowUpRows(patientRows []extract.PatientRow, now time.Time) []extract.YearlyFollowUpRow {
	var rows []extract.YearlyFollowUpRow

	for _, pr := range patientRows {
		// Must have medication (Yes or Other)
		if !pr.Medication.HasMedication() {
			continue
		}

		// Must have a discharge date
		if pr.DischargeDate == "" || pr.DischargeDate == "N/A" {
			continue
		}

		// Parse discharge date (YYYY-MM-DD format)
		dischargeTime, err := time.Parse("2006-01-02", pr.DischargeDate)
		if err != nil {
			continue
		}

		// Include if 9+ months since discharge
		nineMonthsAfter := dischargeTime.AddDate(0, 9, 0)
		if now.Before(nineMonthsAfter) {
			continue
		}

		// Follow-up due date is 12 months after discharge
		followUpDue := dischargeTime.AddDate(1, 0, 0)

		rows = append(rows, extract.YearlyFollowUpRow{
			PatientName:     pr.PatientName,
			ReferenceNumber: pr.ReferenceNumber,
			DischargeDate:   pr.DischargeDate,
			FollowUpDueDate: followUpDue.Format("02/01/2006"),
			Medication:      pr.Medication,
			ReferringGP:     pr.ReferringGP,
		})
	}

	// Sort by follow-up due date ascending
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].FollowUpDueDate < rows[j].FollowUpDueDate
	})

	return rows
}
