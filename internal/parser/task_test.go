package parser

import (
	"strings"
	"testing"

	"github.com/severity1/claude-agent-sdk-go/internal/shared"
)

func TestParseTaskStartedMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		wantErr string
		check   func(t *testing.T, m *shared.TaskStartedMessage)
	}{
		{
			name: "full payload",
			data: taskPayload("task_started", map[string]any{
				"description":     "Explore the parser",
				"tool_use_id":     "toolu_01",
				"task_type":       "local_agent",
				"subagent_type":   "Explore",
				"is_backgrounded": false,
			}),
			check: func(t *testing.T, m *shared.TaskStartedMessage) {
				t.Helper()
				assertTaskString(t, "TaskID", m.TaskID, "task-1")
				assertTaskString(t, "Description", m.Description, "Explore the parser")
				assertTaskString(t, "UUID", m.UUID, "uuid-1")
				assertTaskString(t, "SessionID", m.SessionID, "session-1")
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "toolu_01")
				assertTaskStringPtr(t, "TaskType", m.TaskType, "local_agent")
				assertTaskString(t, "Subtype", m.Subtype, shared.SystemSubtypeTaskStarted)
				if m.Data["subagent_type"] != "Explore" {
					t.Errorf("Data[subagent_type] = %v, want Explore", m.Data["subagent_type"])
				}
			},
		},
		{
			name: "minimal payload",
			data: taskPayload("task_started", map[string]any{"description": "d"}),
			check: func(t *testing.T, m *shared.TaskStartedMessage) {
				t.Helper()
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "")
				assertTaskStringPtr(t, "TaskType", m.TaskType, "")
			},
		},
		{
			name: "optional fields of the wrong type are absent",
			data: taskPayload("task_started", map[string]any{"description": "d", "tool_use_id": 7.0, "task_type": nil}),
			check: func(t *testing.T, m *shared.TaskStartedMessage) {
				t.Helper()
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "")
				assertTaskStringPtr(t, "TaskType", m.TaskType, "")
			},
		},
		{name: "missing task_id", data: without(taskPayload("task_started", map[string]any{"description": "d"}), "task_id"), wantErr: "task_started message missing task_id field"},
		{name: "missing description", data: taskPayload("task_started", nil), wantErr: "task_started message missing description field"},
		{name: "missing uuid", data: without(taskPayload("task_started", map[string]any{"description": "d"}), "uuid"), wantErr: "task_started message missing uuid field"},
		{name: "missing session_id", data: without(taskPayload("task_started", map[string]any{"description": "d"}), "session_id"), wantErr: "task_started message missing session_id field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := parseTaskSystemMessage(t, tt.data, tt.wantErr)
			if sys == nil {
				return
			}
			m, ok := sys.AsTaskStarted()
			if !ok {
				t.Fatal("AsTaskStarted() ok = false, want true")
			}
			tt.check(t, m)
		})
	}
}

func TestParseTaskProgressMessage(t *testing.T) {
	usage := map[string]any{"total_tokens": 1200.0, "tool_uses": 3.0, "duration_ms": 4500.0}
	tests := []struct {
		name    string
		data    map[string]any
		wantErr string
		check   func(t *testing.T, m *shared.TaskProgressMessage)
	}{
		{
			name: "full payload",
			data: taskPayload("task_progress", map[string]any{
				"description":    "Explore the parser",
				"usage":          usage,
				"tool_use_id":    "toolu_01",
				"last_tool_name": "Read",
				"summary":        "reading json.go",
			}),
			check: func(t *testing.T, m *shared.TaskProgressMessage) {
				t.Helper()
				assertTaskString(t, "TaskID", m.TaskID, "task-1")
				assertTaskString(t, "Description", m.Description, "Explore the parser")
				if m.Usage != (shared.TaskUsage{TotalTokens: 1200, ToolUses: 3, DurationMs: 4500}) {
					t.Errorf("Usage = %+v", m.Usage)
				}
				assertTaskString(t, "UUID", m.UUID, "uuid-1")
				assertTaskString(t, "SessionID", m.SessionID, "session-1")
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "toolu_01")
				assertTaskStringPtr(t, "LastToolName", m.LastToolName, "Read")
				if m.Data["summary"] != "reading json.go" {
					t.Errorf("Data[summary] = %v, want reading json.go", m.Data["summary"])
				}
			},
		},
		{
			name: "minimal payload",
			data: taskPayload("task_progress", map[string]any{"description": "d", "usage": map[string]any{}}),
			check: func(t *testing.T, m *shared.TaskProgressMessage) {
				t.Helper()
				if m.Usage != (shared.TaskUsage{}) {
					t.Errorf("Usage = %+v, want zero", m.Usage)
				}
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "")
				assertTaskStringPtr(t, "LastToolName", m.LastToolName, "")
			},
		},
		{name: "missing task_id", data: without(taskPayload("task_progress", map[string]any{"description": "d", "usage": usage}), "task_id"), wantErr: "task_progress message missing task_id field"},
		{name: "missing description", data: taskPayload("task_progress", map[string]any{"usage": usage}), wantErr: "task_progress message missing description field"},
		{name: "missing usage", data: taskPayload("task_progress", map[string]any{"description": "d"}), wantErr: "task_progress message missing usage field"},
		{name: "missing uuid", data: without(taskPayload("task_progress", map[string]any{"description": "d", "usage": usage}), "uuid"), wantErr: "task_progress message missing uuid field"},
		{name: "missing session_id", data: without(taskPayload("task_progress", map[string]any{"description": "d", "usage": usage}), "session_id"), wantErr: "task_progress message missing session_id field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := parseTaskSystemMessage(t, tt.data, tt.wantErr)
			if sys == nil {
				return
			}
			m, ok := sys.AsTaskProgress()
			if !ok {
				t.Fatal("AsTaskProgress() ok = false, want true")
			}
			tt.check(t, m)
		})
	}
}

func TestParseTaskNotificationMessage(t *testing.T) {
	required := func(extra map[string]any) map[string]any {
		fields := map[string]any{"status": "completed", "output_file": "/tmp/out.txt", "summary": "done"}
		for k, v := range extra {
			fields[k] = v
		}
		return taskPayload("task_notification", fields)
	}
	tests := []struct {
		name    string
		data    map[string]any
		wantErr string
		check   func(t *testing.T, m *shared.TaskNotificationMessage)
	}{
		{
			name: "full payload",
			data: required(map[string]any{
				"status":          "stopped",
				"tool_use_id":     "toolu_01",
				"usage":           map[string]any{"total_tokens": 10.0, "tool_uses": 1.0, "duration_ms": 20.0},
				"skip_transcript": true,
			}),
			check: func(t *testing.T, m *shared.TaskNotificationMessage) {
				t.Helper()
				assertTaskString(t, "TaskID", m.TaskID, "task-1")
				if m.Status != shared.TaskNotificationStatusStopped {
					t.Errorf("Status = %q, want stopped", m.Status)
				}
				assertTaskString(t, "OutputFile", m.OutputFile, "/tmp/out.txt")
				assertTaskString(t, "Summary", m.Summary, "done")
				assertTaskString(t, "UUID", m.UUID, "uuid-1")
				assertTaskString(t, "SessionID", m.SessionID, "session-1")
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "toolu_01")
				if m.Usage == nil || *m.Usage != (shared.TaskUsage{TotalTokens: 10, ToolUses: 1, DurationMs: 20}) {
					t.Errorf("Usage = %+v", m.Usage)
				}
				if m.Data["skip_transcript"] != true {
					t.Errorf("Data[skip_transcript] = %v, want true", m.Data["skip_transcript"])
				}
			},
		},
		{
			name: "minimal payload",
			data: required(nil),
			check: func(t *testing.T, m *shared.TaskNotificationMessage) {
				t.Helper()
				if m.Status != shared.TaskNotificationStatusCompleted {
					t.Errorf("Status = %q, want completed", m.Status)
				}
				assertTaskStringPtr(t, "ToolUseID", m.ToolUseID, "")
				if m.Usage != nil {
					t.Errorf("Usage = %+v, want nil", m.Usage)
				}
			},
		},
		{name: "missing task_id", data: without(required(nil), "task_id"), wantErr: "task_notification message missing task_id field"},
		{name: "missing status", data: without(required(nil), "status"), wantErr: "task_notification message missing status field"},
		{name: "missing output_file", data: without(required(nil), "output_file"), wantErr: "task_notification message missing output_file field"},
		{name: "missing summary", data: without(required(nil), "summary"), wantErr: "task_notification message missing summary field"},
		{name: "missing uuid", data: without(required(nil), "uuid"), wantErr: "task_notification message missing uuid field"},
		{name: "missing session_id", data: without(required(nil), "session_id"), wantErr: "task_notification message missing session_id field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := parseTaskSystemMessage(t, tt.data, tt.wantErr)
			if sys == nil {
				return
			}
			m, ok := sys.AsTaskNotification()
			if !ok {
				t.Fatal("AsTaskNotification() ok = false, want true")
			}
			tt.check(t, m)
		})
	}
}

func TestParseTaskUpdatedMessage(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]any
		wantTaskID    string
		wantStatus    string
		wantPatchKeys int
		wantSessionID string
		wantUUID      string
	}{
		{
			name: "full payload",
			data: taskPayload("task_updated", map[string]any{
				"patch": map[string]any{"status": "killed", "end_time": 1759312000000.0},
			}),
			wantTaskID:    "task-1",
			wantStatus:    "killed",
			wantPatchKeys: 2,
			wantSessionID: "session-1",
			wantUUID:      "uuid-1",
		},
		{
			name:          "minimal payload",
			data:          map[string]any{"type": "system", "subtype": "task_updated", "task_id": "task-1", "patch": map[string]any{"status": "running"}},
			wantTaskID:    "task-1",
			wantStatus:    "running",
			wantPatchKeys: 1,
		},
		{
			name:          "patch without status",
			data:          taskPayload("task_updated", map[string]any{"patch": map[string]any{"is_backgrounded": true}}),
			wantTaskID:    "task-1",
			wantPatchKeys: 1,
			wantSessionID: "session-1",
			wantUUID:      "uuid-1",
		},
		{
			name:          "patch that is not an object",
			data:          taskPayload("task_updated", map[string]any{"patch": "killed"}),
			wantTaskID:    "task-1",
			wantSessionID: "session-1",
			wantUUID:      "uuid-1",
		},
		{
			name:          "patch status that is not a string",
			data:          taskPayload("task_updated", map[string]any{"patch": map[string]any{"status": 3.0}}),
			wantTaskID:    "task-1",
			wantPatchKeys: 1,
			wantSessionID: "session-1",
			wantUUID:      "uuid-1",
		},
		{
			name: "missing everything but the subtype",
			data: map[string]any{"type": "system", "subtype": "task_updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys := parseTaskSystemMessage(t, tt.data, "")
			m, ok := sys.AsTaskUpdated()
			if !ok {
				t.Fatal("AsTaskUpdated() ok = false, want true")
			}
			assertTaskString(t, "TaskID", m.TaskID, tt.wantTaskID)
			if m.Patch == nil || len(m.Patch) != tt.wantPatchKeys {
				t.Errorf("Patch = %v, want a map with %d keys", m.Patch, tt.wantPatchKeys)
			}
			gotStatus := ""
			if m.Status != nil {
				gotStatus = string(*m.Status)
			}
			assertTaskString(t, "Status", gotStatus, tt.wantStatus)
			assertTaskStringPtr(t, "SessionID", m.SessionID, tt.wantSessionID)
			assertTaskStringPtr(t, "UUID", m.UUID, tt.wantUUID)
		})
	}
}

func TestParseNonTaskSystemMessageIsNotTyped(t *testing.T) {
	sys := parseTaskSystemMessage(t, map[string]any{"type": "system", "subtype": "init", "task_id": "task-1"}, "")
	if _, ok := sys.AsTaskStarted(); ok {
		t.Error("AsTaskStarted() ok = true for an init message")
	}
	if _, ok := sys.AsTaskUpdated(); ok {
		t.Error("AsTaskUpdated() ok = true for an init message")
	}
}

// parseTaskSystemMessage parses data and returns the SystemMessage, or nil
// after checking the expected parse error.
func parseTaskSystemMessage(t *testing.T, data map[string]any, wantErr string) *shared.SystemMessage {
	t.Helper()
	msg, err := New().ParseMessage(data)
	if wantErr != "" {
		if err == nil || !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("ParseMessage error = %v, want one containing %q", err, wantErr)
		}
		if !shared.IsMessageParseError(err) {
			t.Errorf("ParseMessage error %T is not a MessageParseError", err)
		}
		return nil
	}
	if err != nil {
		t.Fatalf("ParseMessage error = %v", err)
	}
	sys, ok := msg.(*shared.SystemMessage)
	if !ok {
		t.Fatalf("ParseMessage returned %T, want *shared.SystemMessage", msg)
	}
	return sys
}

func taskPayload(subtype string, fields map[string]any) map[string]any {
	data := map[string]any{
		"type":       "system",
		"subtype":    subtype,
		"task_id":    "task-1",
		"uuid":       "uuid-1",
		"session_id": "session-1",
	}
	for k, v := range fields {
		data[k] = v
	}
	return data
}

func without(data map[string]any, key string) map[string]any {
	delete(data, key)
	return data
}

func assertTaskString(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertTaskStringPtr(t *testing.T, field string, got *string, want string) {
	t.Helper()
	switch {
	case want == "" && got != nil:
		t.Errorf("%s = %q, want nil", field, *got)
	case want != "" && (got == nil || *got != want):
		t.Errorf("%s = %v, want %q", field, got, want)
	}
}
