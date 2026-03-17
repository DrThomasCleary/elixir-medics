package report

import (
	"strconv"
	"strings"
	"time"

	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/schani/elixir-medics/internal/extract"
)

// GenerateSubmissionsReport generates statistics for the submissions report.
func GenerateSubmissionsReport(
	appointments []extract.AppointmentWithPatient,
	contacts []extract.PatientContact,
	rows []extract.ExtractedRow,
	allPatients []cliniko.Patient,
	month, year int,
) extract.SubmissionsReport {
	// Calculate end of reporting month
	endOfMonth := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)

	// Filter appointments to the target month for DNA stats
	var monthAppointments []extract.AppointmentWithPatient
	for _, apt := range appointments {
		t, err := time.Parse(time.RFC3339, apt.AppointmentDateTimeRaw)
		if err != nil {
			continue
		}
		if int(t.Month()) == month && t.Year() == year {
			monthAppointments = append(monthAppointments, apt)
		}
	}

	var dnaAppointments int
	for _, apt := range monthAppointments {
		if !apt.Arrived {
			dnaAppointments++
		}
	}

	// DNA percentage is calculated after caseload is known (denominator = caseload)

	// Count patients whose first arrived appointment is in this month
	// Group appointments by patient (referralId)
	appointmentsByPatient := make(map[string][]extract.AppointmentWithPatient)
	for _, apt := range appointments {
		appointmentsByPatient[apt.ReferralID] = append(appointmentsByPatient[apt.ReferralID], apt)
	}

	var newPatientsCount int
	for _, patientApts := range appointmentsByPatient {
		// Sort by date and find first arrived appointment
		sorted := sortAppointmentsByDate(patientApts)
		var firstArrived *extract.AppointmentWithPatient
		for i := range sorted {
			if sorted[i].Arrived {
				firstArrived = &sorted[i]
				break
			}
		}
		if firstArrived != nil {
			t, err := time.Parse(time.RFC3339, firstArrived.AppointmentDateTimeRaw)
			if err == nil && int(t.Month()) == month && t.Year() == year {
				newPatientsCount++
			}
		}
	}

	// Count patient contacts (treatment notes + communications) in the month
	var contactsCount int
	for _, c := range contacts {
		t, err := time.Parse(time.RFC3339, c.CreatedAt)
		if err != nil {
			continue
		}
		if int(t.Month()) == month && t.Year() == year {
			contactsCount++
		}
	}

	// Build a map of patient data from rows (one per patient, using Initial type rows)
	// Key: ReferenceNumber, Value: patient data
	type patientData struct {
		referralDate          string
		dischargeDate         string
		positiveDiagnosis     extract.TriState
		previousDiagnosis     extract.TriState
		medication            extract.MedicationStatus
		sharedCare            extract.TriState
		initialAssessmentDate string // raw ISO date
	}
	patientDataByRef := make(map[string]patientData)
	for _, row := range rows {
		// Only consider Initial type rows (which have the first appointment date)
		if row.Type == extract.TypeInitial {
			patientDataByRef[row.ReferenceNumber] = patientData{
				referralDate:          row.DateOfReferral,
				dischargeDate:         row.DischargeDate,
				positiveDiagnosis:     row.PositiveDiagnosis,
				previousDiagnosis:     row.PreviousDiagnosis,
				medication:            row.Medication,
				sharedCare:            row.SharedCare,
				initialAssessmentDate: row.DateOfAssessmentRaw,
			}
		} else if _, exists := patientDataByRef[row.ReferenceNumber]; !exists {
			// If no Initial row yet, store what we have
			patientDataByRef[row.ReferenceNumber] = patientData{
				referralDate:          row.DateOfReferral,
				dischargeDate:         row.DischargeDate,
				positiveDiagnosis:     row.PositiveDiagnosis,
				previousDiagnosis:     row.PreviousDiagnosis,
				medication:            row.Medication,
				sharedCare:            row.SharedCare,
				initialAssessmentDate: row.DateOfAssessmentRaw,
			}
		}
	}

	// Count referrals from ALL patients (not just those with OldReferenceID)
	var referralsCount, referralsWithPreviousDiagnosis, referralsWithoutPreviousDiagnosis int
	for _, p := range allPatients {
		cf := extract.ParseCustomFields(p)
		if cf.ReferralDate == nil || *cf.ReferralDate == "" {
			continue
		}
		parts := strings.Split(*cf.ReferralDate, "-")
		if len(parts) != 3 {
			continue
		}
		refYear, err1 := strconv.Atoi(parts[0])
		refMonth, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil || refMonth != month || refYear != year {
			continue
		}
		referralsCount++
		prevDiag := extract.BoolToTriState(cf.PreviousDiagnosis)
		if prevDiag == extract.TriStateYes {
			referralsWithPreviousDiagnosis++
		} else {
			referralsWithoutPreviousDiagnosis++
		}
	}

	// Calculate caseload: patients referred but not discharged by end of month
	// - Referral date must be <= end of month (includes patients referred before start of month!)
	// - Discharge date must be empty OR > end of month
	// Also count patients on caseload with medication AND shared care
	var caseloadCount int
	var sharedCareCount int
	for _, pd := range patientDataByRef {
		// Check referral date is before or on end of month
		if pd.referralDate == "" || pd.referralDate == "N/A" {
			continue
		}
		refParts := strings.Split(pd.referralDate, "-")
		if len(refParts) != 3 {
			continue
		}
		refYear, err1 := strconv.Atoi(refParts[0])
		refMonth, err2 := strconv.Atoi(refParts[1])
		refDay, err3 := strconv.Atoi(refParts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		referralTime := time.Date(refYear, time.Month(refMonth), refDay, 0, 0, 0, 0, time.UTC)
		if referralTime.After(endOfMonth) {
			continue // Referred after end of month
		}

		// Check discharge date (if present, must be after end of month)
		if pd.dischargeDate != "" && pd.dischargeDate != "N/A" {
			disParts := strings.Split(pd.dischargeDate, "-")
			if len(disParts) == 3 {
				disYear, e1 := strconv.Atoi(disParts[0])
				disMonth, e2 := strconv.Atoi(disParts[1])
				disDay, e3 := strconv.Atoi(disParts[2])
				if e1 == nil && e2 == nil && e3 == nil {
					dischargeTime := time.Date(disYear, time.Month(disMonth), disDay, 0, 0, 0, 0, time.UTC)
					if !dischargeTime.After(endOfMonth) {
						continue // Discharged before or on end of month
					}
				}
			}
		}

		caseloadCount++

		// Count patients on caseload with medication AND shared care
		// Both "Prescribed" (Yes) and "Other" count as having medication
		if pd.medication.HasMedication() && pd.sharedCare == extract.TriStateYes {
			sharedCareCount++
		}
	}

	var dnaPercentage float64
	dnaPlusInitial := dnaAppointments + newPatientsCount
	if dnaPlusInitial > 0 {
		dnaPercentage = float64(dnaAppointments) / float64(dnaPlusInitial) * 100
	}

	// Patients with initial assessment in month + positive diagnosis + no previous diagnosis
	var psychTherapiesCount int
	for _, pd := range patientDataByRef {
		if pd.initialAssessmentDate == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, pd.initialAssessmentDate)
		if err != nil {
			continue
		}
		if int(t.Month()) == month && t.Year() == year && pd.positiveDiagnosis == extract.TriStateYes && pd.previousDiagnosis != extract.TriStateYes {
			psychTherapiesCount++
		}
	}

	// Calculate new diagnosis count and percentage:
	// Of patients who had initial assessment during reporting month AND no previous diagnosis,
	// count those with positive diagnosis
	var newDiagnosisCount int
	for _, pd := range patientDataByRef {
		if pd.initialAssessmentDate == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, pd.initialAssessmentDate)
		if err != nil {
			continue
		}
		if int(t.Month()) == month && t.Year() == year && pd.previousDiagnosis != extract.TriStateYes && pd.positiveDiagnosis == extract.TriStateYes {
			newDiagnosisCount++
		}
	}

	var newDiagnosisPercentage float64
	if newPatientsCount > 0 {
		newDiagnosisPercentage = float64(newDiagnosisCount) / float64(newPatientsCount) * 100
	}

	// Count initial assessments by mode and titration appointments in the reporting month
	var initialF2FCount, initialRemoteCount, titrationCount int
	for _, row := range rows {
		if row.DateOfAssessmentRaw == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, row.DateOfAssessmentRaw)
		if err != nil {
			continue
		}
		if int(t.Month()) != month || t.Year() != year {
			continue
		}
		switch row.Type {
		case extract.TypeInitial:
			if row.Mode == extract.ModeRemote {
				initialRemoteCount++
			} else {
				initialF2FCount++
			}
		case extract.TypeTitration:
			titrationCount++
		}
	}

	return extract.SubmissionsReport{
		DNACount:                          dnaAppointments,
		DNAPercentage:                     dnaPercentage,
		InitialAssessmentCount:            newPatientsCount,
		ReferralsCount:                    referralsCount,
		ReferralsWithPreviousDiagnosis:    referralsWithPreviousDiagnosis,
		ReferralsWithoutPreviousDiagnosis: referralsWithoutPreviousDiagnosis,
		PatientContactsCount:              contactsCount,
		CaseloadCount:                     caseloadCount,
		PsychologicalTherapiesCount:       psychTherapiesCount,
		NewDiagnosisCount:                 newDiagnosisCount,
		NewDiagnosisPercentage:            newDiagnosisPercentage,
		SharedCareCount:                   sharedCareCount,
		InitialFaceToFaceCount:            initialF2FCount,
		InitialRemoteCount:                initialRemoteCount,
		TitrationCount:                    titrationCount,
	}
}

// sortAppointmentsByDate sorts appointments by their raw date time.
func sortAppointmentsByDate(apts []extract.AppointmentWithPatient) []extract.AppointmentWithPatient {
	result := make([]extract.AppointmentWithPatient, len(apts))
	copy(result, apts)

	// Simple insertion sort (appointments are typically small lists)
	for i := 1; i < len(result); i++ {
		j := i
		for j > 0 {
			t1, _ := time.Parse(time.RFC3339, result[j-1].AppointmentDateTimeRaw)
			t2, _ := time.Parse(time.RFC3339, result[j].AppointmentDateTimeRaw)
			if t1.After(t2) {
				result[j-1], result[j] = result[j], result[j-1]
				j--
			} else {
				break
			}
		}
	}
	return result
}
