// Package ui provides the Fyne-based graphical user interface.
package ui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/schani/elixir-medics/internal/cliniko"
	"github.com/schani/elixir-medics/internal/keychain"
	"github.com/schani/elixir-medics/internal/report"
)

// App represents the main application.
type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window

	// Widgets
	monthSelect *widget.Select
	yearSelect  *widget.Select
	generateBtn *widget.Button
	progressBar *widget.ProgressBar
	statusLabel *widget.Label

	// State
	generating bool
	cancelFunc context.CancelFunc
}

// NewApp creates a new application instance.
func NewApp() *App {
	a := &App{
		fyneApp: app.NewWithID("com.schani.elixir-medics"),
	}
	a.fyneApp.Settings().SetTheme(&customTheme{})
	return a
}

// Run starts the application.
func (a *App) Run() {
	a.mainWindow = a.fyneApp.NewWindow("Elixir Medics")
	a.mainWindow.Resize(fyne.NewSize(680, 440))

	a.setupUI()
	a.mainWindow.ShowAndRun()
}

func (a *App) setupUI() {
	// Month selector
	months := []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	a.monthSelect = widget.NewSelect(months, nil)
	a.monthSelect.SetSelectedIndex(int(time.Now().Month()) - 1)

	// Year selector
	currentYear := time.Now().Year()
	years := make([]string, 5)
	for i := 0; i < 5; i++ {
		years[i] = strconv.Itoa(currentYear - 2 + i)
	}
	a.yearSelect = widget.NewSelect(years, nil)
	a.yearSelect.SetSelected(strconv.Itoa(currentYear))

	// Generate button
	a.generateBtn = widget.NewButton("Generate Report", a.onGenerate)
	a.generateBtn.Importance = widget.HighImportance

	// Progress bar and status
	a.progressBar = widget.NewProgressBar()
	a.progressBar.Hide()
	a.statusLabel = widget.NewLabel("")
	a.statusLabel.Alignment = fyne.TextAlignCenter

	// Settings link (small, at bottom)
	settingsLink := widget.NewHyperlink("API Key Settings", nil)
	settingsLink.OnTapped = a.showSettingsDialog

	// Date selection row using a form-like grid
	dateForm := container.NewHBox(
		widget.NewLabel("Month:"),
		a.monthSelect,
		layout.NewSpacer(),
		widget.NewLabel("Year:"),
		a.yearSelect,
	)

	content := container.NewVBox(
		dateForm,
		container.NewPadded(container.NewCenter(a.generateBtn)),
		a.progressBar,
		a.statusLabel,
		layout.NewSpacer(),
		container.NewCenter(settingsLink),
	)

	a.mainWindow.SetContent(container.NewPadded(content))
}

func (a *App) showSettingsDialog() {
	// Get current API key
	currentKey, err := keychain.GetAPIKey()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read API key: %v", err), a.mainWindow)
		return
	}

	entry := widget.NewPasswordEntry()
	entry.SetPlaceHolder("Enter Cliniko API Key")
	if currentKey != "" {
		entry.SetText(currentKey)
	}

	items := []*widget.FormItem{
		widget.NewFormItem("API Key", entry),
	}

	d := dialog.NewForm("API Key Settings", "Save", "Cancel", items, func(save bool) {
		if save {
			newKey := entry.Text
			if newKey == "" {
				if err := keychain.DeleteAPIKey(); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to delete API key: %v", err), a.mainWindow)
				}
			} else {
				if err := keychain.SetAPIKey(newKey); err != nil {
					dialog.ShowError(fmt.Errorf("Failed to save API key: %v", err), a.mainWindow)
				}
			}
		}
	}, a.mainWindow)
	d.Resize(fyne.NewSize(400, 150))
	d.Show()
}

func (a *App) onGenerate() {
	if a.generating {
		// Cancel current generation
		if a.cancelFunc != nil {
			a.cancelFunc()
		}
		return
	}

	// Get API key
	apiKey, err := keychain.GetAPIKey()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to read API key: %v", err), a.mainWindow)
		return
	}
	if apiKey == "" {
		dialog.ShowInformation("API Key Required", "Please set your Cliniko API key in settings first.", a.mainWindow)
		return
	}

	// Get month/year
	month := a.monthSelect.SelectedIndex() + 1
	year, _ := strconv.Atoi(a.yearSelect.Selected)

	// Start generation
	a.generating = true
	a.generateBtn.SetText("Cancel")
	a.progressBar.Show()
	a.progressBar.SetValue(0)
	a.statusLabel.SetText("Starting...")

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel

	go a.runGeneration(ctx, apiKey, month, year)
}

func (a *App) runGeneration(ctx context.Context, apiKey string, month, year int) {
	log.Println("[DEBUG] runGeneration started")
	defer func() {
		log.Println("[DEBUG] runGeneration defer - updating UI")
		fyne.Do(func() {
			a.generating = false
			a.generateBtn.SetText("Generate Report")
			a.cancelFunc = nil
		})
		log.Println("[DEBUG] runGeneration defer - done")
	}()

	client := cliniko.NewClient(apiKey)
	generator := report.NewGenerator(client)

	var totalPatients int

	opts := report.Options{
		Month: &month,
		Year:  &year,
		OnPatientsFetched: func(count int, total *int) {
			log.Printf("[DEBUG] OnPatientsFetched: count=%d, total=%v", count, total)
			fyne.Do(func() {
				if total != nil {
					totalPatients = *total
					a.progressBar.SetValue(float64(count) / float64(*total) * 0.2)
					a.statusLabel.SetText(fmt.Sprintf("Fetching patients... %d/%d", count, *total))
				} else {
					a.statusLabel.SetText(fmt.Sprintf("Fetching patients... %d", count))
				}
			})
		},
		OnPatientsProcessed: func(processed, total int) {
			log.Printf("[DEBUG] OnPatientsProcessed: %d/%d", processed, total)
			fyne.Do(func() {
				progress := 0.2 + float64(processed)/float64(total)*0.5
				a.progressBar.SetValue(progress)
				a.statusLabel.SetText(fmt.Sprintf("Processing patients... %d/%d", processed, total))
			})
		},
		OnContactsFetched: func(processed, total int) {
			log.Printf("[DEBUG] OnContactsFetched: %d/%d", processed, total)
			fyne.Do(func() {
				progress := 0.7 + float64(processed)/float64(total)*0.3
				a.progressBar.SetValue(progress)
				a.statusLabel.SetText(fmt.Sprintf("Fetching contacts... %d/%d", processed, total))
			})
		},
	}

	log.Println("[DEBUG] Calling generator.Generate...")
	result, err := generator.Generate(ctx, opts)
	log.Printf("[DEBUG] generator.Generate returned, err=%v", err)

	if err != nil {
		if ctx.Err() == context.Canceled {
			fyne.Do(func() {
				a.statusLabel.SetText("Cancelled")
				a.progressBar.Hide()
			})
			return
		}
		fyne.Do(func() {
			a.statusLabel.SetText("Error: " + err.Error())
			a.progressBar.Hide()
			dialog.ShowError(err, a.mainWindow)
		})
		return
	}

	log.Println("[DEBUG] Generation successful, updating UI")
	fyne.Do(func() {
		a.progressBar.SetValue(1)
		a.statusLabel.SetText(fmt.Sprintf("Done! Processed %d patients.", totalPatients))
		a.showSaveDialog(result)
	})
}

func (a *App) showSaveDialog(result *report.Result) {
	log.Println("[DEBUG] showSaveDialog: entered")
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
	log.Printf("[DEBUG] showSaveDialog: prepared %d files", len(files))

	log.Println("[DEBUG] showSaveDialog: about to show folder dialog")
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		log.Printf("[DEBUG] showSaveDialog callback: uri=%v, err=%v", uri, err)
		if err != nil {
			dialog.ShowError(err, a.mainWindow)
			return
		}
		if uri == nil {
			log.Println("[DEBUG] showSaveDialog: user cancelled")
			return // User cancelled
		}

		basePath := uri.Path()
		log.Printf("[DEBUG] showSaveDialog: saving to %s", basePath)
		for name, content := range files {
			path := basePath + "/" + name
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to write %s: %v", name, err), a.mainWindow)
				return
			}
		}

		dialog.ShowInformation("Success", fmt.Sprintf("Saved %d files to %s", len(files), basePath), a.mainWindow)
	}, a.mainWindow)
	log.Println("[DEBUG] showSaveDialog: ShowFolderOpen returned")
}
