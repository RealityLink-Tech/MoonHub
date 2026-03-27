# PWA Chat Streaming & Tool Activity Display

**Date**: 2026-03-27
**Scope**: Connect MoonHub-PWA to MoonHub backend via WebSocket with real-time agent activity streaming

## Problem

MoonHub-PWA has a complete chat UI built with mock data. MoonHub backend has a working agent loop and Pico WebSocket channel, but the agent loop only returns complete responses — there is no real-time visibility into tool calls, thinking, or streaming content. The PWA needs to connect to the backend and display agent activity as it happens.

## Architecture Overview

```
PWA (React) ←→ WebSocket ←→ Pico Channel ←→ Message Bus ←→ Agent Loop
                                         ↑                   ↓
                                    Agent Events     EventEmitter callback
```

The agent loop emits structured events through an optional callback. The channel manager routes these events to the appropriate channel. The Pico channel broadcasts them to connected WebSocket clients using extended Pico protocol message types.

## Backend Changes

### 1. Agent Event System

New types in `pkg/agent/events.go`:

```go
type AgentEventKind string

const (
    EventToolStart    AgentEventKind = "tool_start"
    EventToolEnd      AgentEventKind = "tool_end"
    EventContentStart AgentEventKind = "content_start"
    EventContentChunk AgentEventKind = "content_chunk"
    EventThinking     AgentEventKind = "thinking"
    EventDone         AgentEventKind = "done"
    EventError        AgentEventKind = "error"
)

type AgentEvent struct {
    Kind       AgentEventKind `json:"kind"`
    ToolName   string         `json:"tool_name,omitempty"`
    ToolArgs   map[string]any `json:"tool_args,omitempty"`
    ToolError  string         `json:"tool_error,omitempty"`
    Content    string         `json:"content,omitempty"`
    SessionKey string         `json:"session_key"`
    Channel    string         `json:"channel"`
    ChatID     string         `json:"chat_id"`
    Iteration  int            `json:"iteration,omitempty"`
}

type EventEmitter func(AgentEvent)
```

### 2. Agent Loop Integration

`AgentLoop` gets an optional `EventEmitter` field. In `runLLMIteration`, emit events at these points:

1. **Before tool execution** (`EventToolStart`): tool name, truncated args preview
2. **After tool execution** (`EventToolEnd`): tool name, success/error status
3. **Before final LLM call** (`EventContentStart`): signals response generation beginning
4. **On thinking content** (`EventThinking`): reasoning content from LLM
5. **On completion** (`EventDone`): final content summary
6. **On error** (`EventError`): error message

The emitter is nil-safe — if no emitter is set, events are silently discarded. This preserves backward compatibility with all existing channels.

### 3. Channel Manager Bridge

`ChannelManager` gets a new method:

```go
func (m *Manager) HandleAgentEvent(ctx context.Context, event AgentEvent)
```

This looks up the channel by `event.Channel` and, if it implements `AgentEventEmitter`, calls `EmitAgentEvent(ctx, event.ChatID, event)`.

New interface in `pkg/channels/interfaces.go`:

```go
type AgentEventEmitter interface {
    EmitAgentEvent(ctx context.Context, chatID string, event agent.AgentEvent) error
}
```

### 4. Pico Protocol Extension

New message types in `pkg/channels/pico/protocol.go`:

```go
TypeAgentToolStart = "agent.tool_start"
TypeAgentToolEnd   = "agent.tool_end"
TypeAgentContent   = "agent.content"
TypeAgentThinking  = "agent.thinking"
TypeAgentDone      = "agent.done"
TypeAgentError     = "agent.error"
```

Payloads:
- `agent.tool_start`: `{tool_name, tool_args_preview}`
- `agent.tool_end`: `{tool_name, success, error}`
- `agent.content`: `{content, done}`
- `agent.thinking`: `{content}`
- `agent.done`: `{content}`
- `agent.error`: `{message}`

Pico channel implements `AgentEventEmitter` by mapping `AgentEvent` to Pico messages and broadcasting to matching sessions.

### 5. Wiring in Gateway Startup

In the gateway command startup, after creating the AgentLoop and ChannelManager:
1. Set the `EventEmitter` on the AgentLoop to call `ChannelManager.HandleAgentEvent`
2. This connects the agent's internal events to all connected channels

## Frontend Changes

### 1. MoonHubClient WebSocket Integration

Update `src/services/device.ts` to:
- Add WebSocket connection management (connect, disconnect, reconnect)
- Handle all Pico protocol message types including new `agent.*` types
- Expose event callbacks: `onToolStart`, `onToolEnd`, `onContent`, `onThinking`, `onDone`, `onError`
- Support abort controller for cancellation

### 2. Chat Store Updates (`src/stores/chat.ts`)

Add to the Zustand store:
- `toolStatus: {name: string, status: 'running' | 'done' | 'error'} | null` — current active tool
- `isStreaming: boolean` — whether agent is actively responding
- `streamContent: string` — accumulated streaming content
- `setToolStatus(name, status)` — update tool indicator
- `appendStreamContent(chunk)` — append streaming text
- `finalizeStream()` — convert stream to final message
- `clearStream()` — reset streaming state

### 3. Chat Page Refactor (`src/pages/Chat.tsx`)

Replace mock implementation:
- Connect to MoonHubClient WebSocket on mount
- Send messages via `message.send` protocol
- Handle incoming `agent.*` events to update UI
- Remove all mock data and setTimeout simulations

### 4. Tool Status Indicator Component

New `src/components/chat/ToolStatusIndicator.tsx`:
- Simple pill/badge below the input area or above the streaming message
- Maps tool names to friendly labels:
  - `web_search` → "Searching the web..."
  - `web_fetch` → "Reading page..."
  - `file_read` → "Reading file..."
  - `send_file` → "Sending file..."
  - `memory` → "Accessing memory..."
  - Default: `"Running {tool_name}..."`
- Shows spinner while running, checkmark on done, X on error
- Auto-hides after tool completes (brief delay)

### 5. StreamingMessage Component Update

Update `src/components/chat/StreamingMessage.tsx`:
- Display accumulated stream content with blinking cursor
- Handle markdown rendering for the final content
- Support the `agent.content` chunks

### 6. Conversation List Integration

Wire `src/components/chat/ConversationList.tsx`:
- Load sessions from `GET /api/sessions` (already exists in backend)
- Create new conversation on first message
- Delete sessions via `DELETE /api/sessions/{id}`

## Message Flow (Complete Example)

```
1. User types "What's the weather in Tokyo?"
2. PWA → WS: message.send {content: "What's the weather in Tokyo?"}
3. Agent receives via bus, starts processing
4. Backend → WS: agent.tool_start {tool_name: "web_search", tool_args_preview: "Tokyo weather"}
5. PWA shows: 🔍 Searching the web...
6. Agent executes web_search tool
7. Backend → WS: agent.tool_end {tool_name: "web_search", success: true}
8. PWA shows: ✓ briefly, then hides
9. Agent calls LLM with search results
10. Backend → WS: agent.content {content: "Based on my search", done: false}
11. PWA appends text to message with cursor
12. Backend → WS: agent.content {content: ", Tokyo is currently sunny...", done: false}
13. PWA appends more text
14. Backend → WS: agent.content {content: "", done: true}
15. PWA finalizes message, removes cursor
16. Backend → WS: agent.done {content: "full response text"}
```

## Files Changed

### Backend (MoonHub)
- `pkg/agent/events.go` — NEW: event types and emitter
- `pkg/agent/loop.go` — MODIFY: emit events in runLLMIteration
- `pkg/agent/instance.go` — MODIFY: add EventEmitter to AgentInstance
- `pkg/channels/interfaces.go` — MODIFY: add AgentEventEmitter interface
- `pkg/channels/manager.go` — MODIFY: add HandleAgentEvent method
- `pkg/channels/pico/protocol.go` — MODIFY: add new message types
- `pkg/channels/pico/pico.go` — MODIFY: implement AgentEventEmitter
- `cmd/moonhub/internal/gateway/` — MODIFY: wire emitter to manager

### Frontend (MoonHub-PWA)
- `src/services/device.ts` — MODIFY: add WebSocket connection management
- `src/stores/chat.ts` — MODIFY: add streaming and tool status state
- `src/pages/Chat.tsx` — MODIFY: replace mock with real WebSocket
- `src/components/chat/ToolStatusIndicator.tsx` — NEW: tool status pill
- `src/components/chat/StreamingMessage.tsx` — MODIFY: handle real streaming
- `src/components/chat/ConversationList.tsx` — MODIFY: wire to real API

## Out of Scope
- Token-by-token LLM streaming (requires provider-level streaming support)
- File/media upload via WebSocket (use existing REST endpoints)
- Session management persistence on PWA side (use existing IndexedDB storage)
- Thinking/reasoning content display (infrastructure added but UI deferred)
