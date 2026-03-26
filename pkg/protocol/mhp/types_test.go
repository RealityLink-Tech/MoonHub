package mhp

import (
	"encoding/json"
	"testing"
)

func TestMessageType_String(t *testing.T) {
	tests := []struct {
		mt       MessageType
		expected string
	}{
		{MsgFriendRequest, "friend_request"},
		{MsgFriendAccept, "friend_accept"},
		{MsgTaskRequest, "task_request"},
		{MsgTaskResponse, "task_response"},
		{MsgFileOffer, "file_offer"},
		{MsgDiscoveryProbe, "discovery_probe"},
		{"custom_type", "custom_type"},
	}
	for _, tt := range tests {
		if got := tt.mt.String(); got != tt.expected {
			t.Errorf("MessageType(%q).String() = %q, want %q", tt.mt, got, tt.expected)
		}
	}
}

func TestTaskRequestPayload_Marshal(t *testing.T) {
	p := TaskRequestPayload{
		Task:      "Write an API",
		Zone:      "shared",
		Resources: []string{"project-code"},
		Priority:  "normal",
		Timeout:   300,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var decoded TaskRequestPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Task != "Write an API" {
		t.Errorf("expected 'Write an API', got %s", decoded.Task)
	}
	if decoded.Priority != "normal" {
		t.Errorf("expected 'normal', got %s", decoded.Priority)
	}
}

func TestFriendRequestPayload_Marshal(t *testing.T) {
	p := FriendRequestPayload{
		AgentName: "TestAgent",
		Message:   "Let's collaborate!",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var decoded FriendRequestPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.AgentName != "TestAgent" {
		t.Errorf("expected TestAgent, got %s", decoded.AgentName)
	}
}

func TestFileOfferPayload_Marshal(t *testing.T) {
	p := FileOfferPayload{
		FileName: "code.py",
		FileSize: 1024,
		Hash:     "abc123",
		Zone:     "shared",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var decoded FileOfferPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.FileName != "code.py" {
		t.Errorf("expected code.py, got %s", decoded.FileName)
	}
}

func TestIsFriendMessageType(t *testing.T) {
	tests := []struct {
		mt       MessageType
		expected bool
	}{
		{MsgFriendRequest, true},
		{MsgFriendAccept, true},
		{MsgFriendReject, true},
		{MsgFriendRevoke, true},
		{MsgFriendPing, true},
		{MsgTaskRequest, false},
		{MsgFileOffer, false},
	}
	for _, tt := range tests {
		if got := tt.mt.IsFriendMessage(); got != tt.expected {
			t.Errorf("IsFriendMessage(%q) = %v, want %v", tt.mt, got, tt.expected)
		}
	}
}
