package common

import (
	"os"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const debugSessionAgentPath = "/Users/mastercui.eth/GitHub/new-api-offical/.cursor/debug-1b0c95.log"

// DebugSessionAgentLog appends one NDJSON line for agent debug mode (session 1b0c95).
// #region agent log
func DebugSessionAgentLog(location, hypothesisID, message string, data map[string]any) {
	f, err := os.OpenFile(debugSessionAgentPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	payload := map[string]any{
		"sessionId":    "1b0c95",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := common.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}

// #endregion
