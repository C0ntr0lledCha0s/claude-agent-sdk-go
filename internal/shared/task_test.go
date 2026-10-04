package shared

import (
	"encoding/json"
	"testing"
)

func TestIsTerminalTaskStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{string(TaskNotificationStatusCompleted), true},
		{string(TaskNotificationStatusFailed), true},
		{string(TaskNotificationStatusStopped), true},
		{string(TaskUpdatedStatusKilled), true},
		{string(TaskUpdatedStatusPending), false},
		{string(TaskUpdatedStatusRunning), false},
		{string(TaskUpdatedStatusPaused), false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := IsTerminalTaskStatus(tt.status); got != tt.want {
				t.Errorf("IsTerminalTaskStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestTaskAccessorsRejectOtherSubtypes(t *testing.T) {
	msg := &SystemMessage{Subtype: "init", Data: map[string]any{
		"type": "system", "subtype": "init", "task_id": "task-1", "description": "d",
		"uuid": "u", "session_id": "s", "usage": map[string]any{}, "status": "completed",
		"output_file": "f", "summary": "s",
	}}

	tests := []struct {
		name string
		ok   bool
	}{
		{"AsTaskStarted", second(msg.AsTaskStarted())},
		{"AsTaskProgress", second(msg.AsTaskProgress())},
		{"AsTaskNotification", second(msg.AsTaskNotification())},
		{"AsTaskUpdated", second(msg.AsTaskUpdated())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ok {
				t.Errorf("%s() ok = true for subtype init", tt.name)
			}
		})
	}
}

func TestTaskMessageKeepsSystemMessage(t *testing.T) {
	data := map[string]any{
		"type": "system", "subtype": SystemSubtypeTaskStarted, "task_id": "task-1",
		"description": "d", "uuid": "u", "session_id": "s", "subagent_type": "Explore",
	}
	sys := &SystemMessage{Subtype: SystemSubtypeTaskStarted, Data: data}

	started, ok := sys.AsTaskStarted()
	if !ok {
		t.Fatal("AsTaskStarted() ok = false")
	}
	var msg Message = started
	if msg.Type() != MessageTypeSystem {
		t.Errorf("Type() = %q, want %q", msg.Type(), MessageTypeSystem)
	}

	out, err := json.Marshal(started)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(out, &round); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if round["subagent_type"] != "Explore" || round["task_id"] != "task-1" {
		t.Errorf("marshaled %s, want the raw payload", out)
	}
}

func second[T any](_ T, ok bool) bool {
	return ok
}
