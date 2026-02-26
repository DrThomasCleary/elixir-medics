package extract

import "strings"

// ParseGP extracts GP information from patient notes.
// Finds a line starting with "GP" (case insensitive) and returns all subsequent lines.
func ParseGP(notes *string) string {
	if notes == nil || *notes == "" {
		return "N/A"
	}

	cleaned := strings.ReplaceAll(*notes, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	lines := strings.Split(cleaned, "\n")
	foundGPLine := false
	var resultLines []string

	for _, line := range lines {
		if foundGPLine {
			resultLines = append(resultLines, line)
		} else if strings.HasPrefix(strings.TrimSpace(strings.ToLower(line)), "gp") {
			// Found the GP line - ignore it, start capturing from next line
			foundGPLine = true
		}
	}

	if len(resultLines) > 0 {
		result := strings.TrimSpace(strings.Join(resultLines, "\n"))
		if result != "" {
			return result
		}
	}

	return "N/A"
}
