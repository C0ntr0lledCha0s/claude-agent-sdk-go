package control

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProtocolStopTask(t *testing.T) {
	tests := []struct {
		name     string
		respond  func(m *controlMockTransport, requestID string)
		wantErr  string
		wantWire string
	}{
		{
			name:     "success",
			respond:  func(m *controlMockTransport, id string) { m.injectResponse(id, nil) },
			wantWire: `{"subtype":"stop_task","task_id":"task-abc123"}`,
		},
		{
			name:     "error_response",
			respond:  func(m *controlMockTransport, id string) { m.injectErrorResponse(id, "no such task") },
			wantErr:  "no such task",
			wantWire: `{"subtype":"stop_task","task_id":"task-abc123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := setupControlTestContext(t, 5*time.Second)
			defer cancel()

			transport := newControlMockTransport()
			protocol := NewProtocol(transport)
			assertControlNoError(t, protocol.Start(ctx))
			defer func() { _ = protocol.Close() }()

			wire := make(chan string, 1)
			go func() {
				data, ok := transport.waitForWrite(time.Now().Add(4*time.Second), func([]byte) bool { return true })
				if !ok {
					wire <- ""
					return
				}
				var raw struct {
					RequestID string          `json:"request_id"`
					Request   json.RawMessage `json:"request"`
				}
				_ = json.Unmarshal(data, &raw)
				wire <- string(raw.Request)
				tt.respond(transport, raw.RequestID)
			}()

			err := protocol.StopTask(ctx, "task-abc123")

			assertStopTaskError(t, err, tt.wantErr)
			if got := <-wire; got != tt.wantWire {
				t.Errorf("request = %s, want %s", got, tt.wantWire)
			}
		})
	}
}

func assertStopTaskError(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		assertControlNoError(t, err)
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("StopTask error = %v, want one containing %q", err, want)
	}
}
