package debuglog

import (
	"encoding/json"
	"os"
	"time"
)

const logPath = "/Users/bauanrashid/Software/elixir-medics/.cursor/debug-04f264.log"

type entry struct {
	SessionID    string         `json:"sessionId"`
	RunID        string         `json:"runId"`
	HypothesisID string         `json:"hypothesisId"`
	Location     string         `json:"location"`
	Message      string         `json:"message"`
	Data         map[string]any `json:"data"`
	Timestamp    int64          `json:"timestamp"`
}

func Log(runID, hypothesisID, location, message string, data map[string]any) {
	b, err := json.Marshal(entry{
		SessionID:    "04f264",
		RunID:        runID,
		HypothesisID: hypothesisID,
		Location:     location,
		Message:      message,
		Data:         data,
		Timestamp:    time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}
