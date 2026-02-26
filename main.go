// Package main provides the entry point for elixir-medics.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/schani/elixir-medics/internal/report"
	"github.com/schani/elixir-medics/internal/ui"
)

func main() {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Parse command-line flags
	cli := flag.Bool("cli", false, "Run in CLI mode (default is GUI)")
	month := flag.Int("month", 0, "Filter to only include rows from this month (1-12)")
	year := flag.Int("year", 0, "Filter to only include rows from this year")
	outputDir := flag.String("output-dir", ".", "Directory to write output files")
	flag.Parse()

	// Auto-detect CLI mode if month/year specified
	useCLI := *cli || (*month != 0 && *year != 0)

	if useCLI {
		runCLI(*month, *year, *outputDir)
	} else {
		runGUI()
	}
}

func runGUI() {
	app := ui.NewApp()
	app.Run()
}

func runCLI(month, year int, outputDir string) {
	// Get API key from environment
	apiKey := os.Getenv("CLINIKO_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: CLINIKO_API_KEY environment variable is required")
		os.Exit(1)
	}

	// Validate month/year
	if (month != 0 && year == 0) || (month == 0 && year != 0) {
		fmt.Fprintln(os.Stderr, "Error: Both --month and --year must be specified together")
		os.Exit(1)
	}

	if month != 0 && (month < 1 || month > 12) {
		fmt.Fprintln(os.Stderr, "Error: --month must be between 1 and 12")
		os.Exit(1)
	}

	// Create client and generator
	client := cliniko.NewClient(apiKey)
	generator := report.NewGenerator(client)

	// Set up options
	opts := report.Options{
		OnPatientsFetched: func(count int, total *int) {
			if total != nil {
				fmt.Fprintf(os.Stderr, "\rFetching patients... %d/%d", count, *total)
			} else {
				fmt.Fprintf(os.Stderr, "\rFetching patients... %d", count)
			}
		},
		OnPatientsProcessed: func(processed, total int) {
			fmt.Fprintf(os.Stderr, "\rProcessing patients... %d/%d", processed, total)
		},
	}

	if month != 0 && year != 0 {
		opts.Month = &month
		opts.Year = &year
	}

	// Generate report
	fmt.Fprintln(os.Stderr, "Starting report generation...")
	ctx := context.Background()
	result, err := generator.Generate(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "\nDone!")

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write output files
	files := map[string]string{
		"invoice.csv":          result.InvoiceCSV,
		"appointments.csv":     result.AppointmentsCSV,
		"patients.csv":         result.PatientsCSV,
		"yearly_follow_up.csv": result.YearlyFollowUpCSV,
	}

	if result.SubmissionsCSV != nil {
		files["submissions.csv"] = *result.SubmissionsCSV
	}
	if result.MonthlyAppointmentsCSV != nil {
		files["appointments_monthly.csv"] = *result.MonthlyAppointmentsCSV
	}

	for filename, content := range files {
		path := filepath.Join(outputDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", path)
	}
}
