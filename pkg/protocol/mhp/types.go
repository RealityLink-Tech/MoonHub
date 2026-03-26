// Package mhp implements the MoonHub Protocol for agent-to-agent communication.
package mhp

// MessageType identifies the type of an mhp message.
type MessageType string

// Friend management message types.
const (
	MsgFriendRequest MessageType = "friend_request"
	MsgFriendAccept  MessageType = "friend_accept"
	MsgFriendReject  MessageType = "friend_reject"
	MsgFriendRevoke  MessageType = "friend_revoke"
	MsgFriendPing    MessageType = "friend_ping"
)

// Task delegation message types.
const (
	MsgTaskRequest  MessageType = "task_request"
	MsgTaskResponse MessageType = "task_response"
	MsgTaskProgress MessageType = "task_progress"
	MsgTaskCancel   MessageType = "task_cancel"
	MsgTaskError    MessageType = "task_error"
)

// File transfer message types.
const (
	MsgFileOffer    MessageType = "file_offer"
	MsgFileAccept   MessageType = "file_accept"
	MsgFileReject   MessageType = "file_reject"
	MsgFileChunk    MessageType = "file_chunk"
	MsgFileComplete MessageType = "file_complete"
)

// Discovery message types.
const (
	MsgDiscoveryProbe    MessageType = "discovery_probe"
	MsgDiscoveryAnnounce MessageType = "discovery_announce"
)

func (mt MessageType) String() string { return string(mt) }

// IsFriendMessage returns true if the message type is a friend management message.
func (mt MessageType) IsFriendMessage() bool {
	switch mt {
	case MsgFriendRequest, MsgFriendAccept, MsgFriendReject, MsgFriendRevoke, MsgFriendPing:
		return true
	}
	return false
}

// FriendRequestPayload is the payload for a friend_request message.
type FriendRequestPayload struct {
	AgentName string `json:"agentName"`
	Message   string `json:"message,omitempty"`
}

// FriendAcceptPayload is the payload for a friend_accept message.
type FriendAcceptPayload struct {
	AgentName string `json:"agentName"`
	PublicKey []byte `json:"publicKey"`
}

// FriendRejectPayload is the payload for a friend_reject message.
type FriendRejectPayload struct {
	Reason string `json:"reason,omitempty"`
}

// TaskRequestPayload is the payload for a task_request message.
type TaskRequestPayload struct {
	Task      string   `json:"task"`
	Zone      string   `json:"zone"`
	Resources []string `json:"resources,omitempty"`
	Priority  string   `json:"priority"`
	Timeout   int      `json:"timeout"`
	ReplyTo   string   `json:"replyTo,omitempty"`
}

// TaskResponsePayload is the payload for a task_response message.
type TaskResponsePayload struct {
	TaskID      string   `json:"taskId"`
	Result      string   `json:"result"`
	Attachments []string `json:"attachments,omitempty"`
}

// TaskProgressPayload is the payload for a task_progress message.
type TaskProgressPayload struct {
	TaskID  string  `json:"taskId"`
	Content string  `json:"content"`
	Percent float64 `json:"percent,omitempty"`
}

// FileOfferPayload is the payload for a file_offer message.
type FileOfferPayload struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	Hash     string `json:"hash"`
	Zone     string `json:"zone"`
}

// FileAcceptPayload is the payload for a file_accept message.
type FileAcceptPayload struct {
	FileName string `json:"fileName"`
}

// FileRejectPayload is the payload for a file_reject message.
type FileRejectPayload struct {
	Reason string `json:"reason,omitempty"`
}

// FileChunkPayload is the payload for a file_chunk message.
type FileChunkPayload struct {
	FileName string `json:"fileName"`
	Offset   int64  `json:"offset"`
	Data     []byte `json:"data"`
}

// FileCompletePayload is the payload for a file_complete message.
type FileCompletePayload struct {
	FileName string `json:"fileName"`
	Hash     string `json:"hash"`
}
