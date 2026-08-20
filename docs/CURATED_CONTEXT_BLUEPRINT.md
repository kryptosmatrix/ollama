# Curated Context System — Implementation Blueprint (v10)

| Field | Value |
|---|---|
| Status | `[BLUEPRINT-DRAFT v10]` — 1 Cursor round-9 finding fixed, pending re-review |
| Document class | Implementation blueprint |
| Author | Fathom (GLM-5.2) |
| Date | 2026-08-19, Australia/Brisbane |
| Repository | github.com/ollama/ollama (Ash's fork) |
| Concept authority | `docs/CURATED_CONTEXT_SYSTEM.md` (design document) |
| Implementation authority | None — pending judge pass |
| Compatibility | Alongside existing compactor; toggle via settings |
| Review history | v1: 6 BLOCKING; v2: 14 BLOCKING; v3: 7 BLOCKING; v4: 7 BLOCKING; v5: 5 BLOCKING; v6: 5+ BLOCKING; v7: 4 BLOCKING; v8: 4 BLOCKING; v9: 1 BLOCKING |
| Source verification | Read actual agent/session.go, cmd/agent_tui.go, cmd/tui/chat/chat.go, LETHE MemoryTools.swift, MCPServer.swift, StdioJSONRPC.swift |

---

## 0. What changed in v10 and why

v10 fixes the 1 finding from Cursor round 9:

| Finding | Fix |
|---|---|
| B7: CurateForRun no-ops because ChatID is never set on TUI path | CurateForRun uses "default" when chatID is empty; StorePostResponse uses same default; AgentName is still required |

### What v9 closed (confirmed by reviewer, not re-litigated)

- Scopes parse: `{"scopes":[...]}` correctly parsed ✓
- AgentName/Curator wiring: all structs have fields, all copy sites specified ✓
- stripMemoryBlocks: works after compaction, no panic ✓
- Compactor composition: stripMemoryBlocks runs at end ✓

---

## 1. Purpose

Replace episodic replay with semantic working memory. The Curator curates messages BEFORE the model sees them (pre-model, in buildRunMessages). The existing Compactor (SimpleCompactor) is UNCHANGED and handles post-model tool-overflow recovery.

**Key lifecycle guarantee:** The model sees curated messages. Persistence uses `stripMemoryBlocks` to remove memory blocks before returning. Memory blocks never reach the persisted conversation history.

---

## 2. Affected files

### 2.1 New files

| File | Purpose |
|---|---|
| `agent/curator.go` | Curator struct, CurateForRun, StorePostResponse, curation logic |
| `agent/lethe_mcp_client.go` | MCP stdio JSON-RPC client for LETHE (store, recall, scopes) |
| `agent/chunker.go` | Semantic chunking of messages |
| `agent/synthesizer.go` | Synthesize retrieved chunks into narrative with pointers |
| `cmd/config/context_management.go` | ContextManagementSettings struct and loader |

### 2.2 Modified files

| File | Change |
|---|---|
| `agent/session.go` | Add Curator field to Session; call CurateForRun in buildRunMessages BEFORE checkPreflightPromptBudget; call StorePostResponse in finishRun; call stripMemoryBlocks in finishRun; add AgentName to RunOptions |
| `cmd/agent_tui.go` | Add Curator and AgentName to agentTUIOptions; read ContextManagementSettings; construct Curator when enabled; use cmd.Context(); expand ~ via os.UserHomeDir(); apply loaded settings; require AgentName |
| `cmd/tui/chat/chat.go` | Add Curator and AgentName fields to Options struct; pass to Session and RunOptions |
| `agent/compactor.go` | No change |
| `server/routes.go` | No change |

---

## 3. Prerequisites

### 3.1 LETHE rawText consent

```bash
lethe-memory-mcp grant --scope connected:ollama --mode readwrite --raw-text
```

The Curator's `CheckConsent` method (§4.1) verifies this by calling the `scopes` tool and checking `rawText: true`. If not granted, the Curator logs a warning and the TUI falls back to the existing Compactor.

### 3.2 LETHE MCP server path

Default: `~/.local/bin/lethe-memory-mcp`. Expanded via `os.UserHomeDir()` before `exec.Command`.

---

## 4. Types and interfaces

### 4.1 Curator (agent/curator.go)

```go
package agent

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "github.com/ollama/ollama/api"
)

type Curator struct {
    Client      ChatClient
    LetheClient *LetheMCPClient
    Chunker     *Chunker
    Synthesizer *Synthesizer
    Options     CuratorOptions
    mu          sync.Mutex
}

type CuratorOptions struct {
    SummaryInterval     int
    KeepRecentTurns     int
    KeepRelevantChunks  int
    SynthesizeNarrative bool
    StoreMessagesInLethe bool
    Scope               string
}

// CurateForRun curates messages BEFORE the model sees them.
// B7a fix: takes agentName and chatID from opts.
// v10 fix: uses "default" when chatID is empty so single-conversation
// TUI use works without threading ChatID through the entire TUI.
// AgentName is still required — it provides agent isolation.
func (c *Curator) CurateForRun(ctx context.Context, messages []api.Message, agentName, chatID, model string) ([]api.Message, error) {
    if agentName == "" {
        return messages, nil // AgentName required for isolation
    }
    // v10 fix: use "default" when chatID is empty
    if chatID == "" {
        chatID = "default"
    }

    summaries, err := c.retrieveSummaries(ctx, agentName, chatID)
    if err != nil {
        return nil, fmt.Errorf("curator retrieve summaries: %w", err)
    }

    chunks, err := c.searchRelevantChunks(ctx, messages, agentName, chatID)
    if err != nil {
        return nil, fmt.Errorf("curator search chunks: %w", err)
    }

    var narrative string
    if c.Options.SynthesizeNarrative && len(chunks) > 0 {
        narrative, err = c.Synthesizer.Synthesize(ctx, model, chunks)
        if err != nil {
            slog.Warn("curator synthesis failed", "error", err)
            narrative = ""
        }
    }

    return c.constructCuratedMessages(messages, summaries, narrative), nil
}

// retrieveSummaries retrieves summaries from LETHE.
// Uses content when available (rawText granted), falls back to summary
// (nullable — null unmarshals to "" in Go), then skips if both empty.
func (c *Curator) retrieveSummaries(ctx context.Context, agentName, chatID string) (map[string]string, error) {
    summaryTypes := []string{"persona", "events", "interests", "topics", "important"}
    summaries := make(map[string]string)

    for _, st := range summaryTypes {
        query := fmt.Sprintf("%s %s %s", agentName, chatID, st)
        hits, err := c.LetheClient.Recall(ctx, c.Options.Scope, query, 10)
        if err != nil {
            return nil, err
        }

        prefix := fmt.Sprintf("[%s %s %s]", agentName, chatID, st)
        for _, hit := range hits {
            text := hit.Content
            if text == "" {
                text = hit.Summary
            }
            if text == "" {
                continue
            }
            if strings.HasPrefix(text, prefix) {
                summaries[st] = strings.TrimPrefix(text, prefix+" ")
                break
            }
        }
    }

    return summaries, nil
}

// searchRelevantChunks searches LETHE for relevant chunks.
func (c *Curator) searchRelevantChunks(ctx context.Context, messages []api.Message, agentName, chatID string) ([]LetheHit, error) {
    latestMessage := findLastUserMessage(messages)
    if latestMessage == "" {
        return nil, nil
    }

    query := fmt.Sprintf("%s %s %s", agentName, chatID, latestMessage)
    hits, err := c.LetheClient.Recall(ctx, c.Options.Scope, query, c.Options.KeepRelevantChunks)
    if err != nil {
        return nil, err
    }

    prefix := fmt.Sprintf("[%s %s]", agentName, chatID)
    var filtered []LetheHit
    for _, hit := range hits {
        text := hit.Content
        if text == "" {
            text = hit.Summary
        }
        if text != "" && strings.HasPrefix(text, prefix) {
            filtered = append(filtered, hit)
        }
    }

    return filtered, nil
}

// constructCuratedMessages builds the curated message list.
// Memory blocks use user role with escaped boundary markers.
func (c *Curator) constructCuratedMessages(messages []api.Message, summaries map[string]string, narrative string) []api.Message {
    var result []api.Message

    if len(messages) > 0 && messages[0].Role == "system" {
        result = append(result, messages[0])
    }

    summaryContent := c.formatSummaries(summaries)
    if summaryContent != "" {
        result = append(result, api.Message{
            Role:    "user",
            Content: "<memory_context>\n" + escapeBoundaryTags(summaryContent) + "\n</memory_context>",
        })
    }

    if narrative != "" {
        result = append(result, api.Message{
            Role:    "user",
            Content: "<relevant_memories>\n" + escapeBoundaryTags(narrative) + "\n</relevant_memories>",
        })
    }

    recentMsgs := sliceRecentTurns(messages, c.Options.KeepRecentTurns)
    result = append(result, recentMsgs...)

    return result
}

// formatSummaries returns empty string when all summaries are empty.
func (c *Curator) formatSummaries(summaries map[string]string) string {
    var sb strings.Builder
    hasContent := false

    if s, ok := summaries["persona"]; ok && s != "" {
        sb.WriteString("## Who you are\n"); sb.WriteString(s); sb.WriteString("\n\n")
        hasContent = true
    }
    if s, ok := summaries["events"]; ok && s != "" {
        sb.WriteString("## What happened\n"); sb.WriteString(s); sb.WriteString("\n\n")
        hasContent = true
    }
    if s, ok := summaries["interests"]; ok && s != "" {
        sb.WriteString("## Your interests\n"); sb.WriteString(s); sb.WriteString("\n\n")
        hasContent = true
    }
    if s, ok := summaries["topics"]; ok && s != "" {
        sb.WriteString("## Current topic\n"); sb.WriteString(s); sb.WriteString("\n\n")
        hasContent = true
    }
    if s, ok := summaries["important"]; ok && s != "" {
        sb.WriteString("## Important items\n"); sb.WriteString(s); sb.WriteString("\n\n")
        hasContent = true
    }

    if !hasContent {
        return ""
    }
    return "Conversation context:\n\n" + sb.String()
}

func (c *Curator) shouldGenerateSummaries(messageCount int) bool {
    if c.Options.SummaryInterval <= 0 {
        return false
    }
    return messageCount > 0 && messageCount % c.Options.SummaryInterval == 0
}

func (c *Curator) generateSummaries(ctx context.Context, messages []api.Message, agentName, chatID, model string) error {
    // v10 fix: use "default" when chatID is empty
    if chatID == "" {
        chatID = "default"
    }
    summaryTypes := []string{"persona", "events", "interests", "topics", "important"}
    for _, st := range summaryTypes {
        prompt := c.buildSummaryPrompt(messages, st)
        summary, err := c.callModel(ctx, model, prompt)
        if err != nil {
            return fmt.Errorf("generate summary %s: %w", st, err)
        }
        prefix := fmt.Sprintf("[%s %s %s] ", agentName, chatID, st)
        if err := c.LetheClient.Store(ctx, c.Options.Scope,
            []string{agentName, chatID, st}, prefix+summary); err != nil {
            return fmt.Errorf("store summary %s: %w", st, err)
        }
    }
    return nil
}

// StorePostResponse stores user and assistant messages in LETHE.
// v10 fix: uses "default" when chatID is empty.
func (c *Curator) StorePostResponse(ctx context.Context, userMsg, assistantMsg api.Message, messageIndex int, agentName, chatID string) {
    if !c.Options.StoreMessagesInLethe {
        return
    }
    if agentName == "" {
        return
    }
    // v10 fix: use "default" when chatID is empty
    if chatID == "" {
        chatID = "default"
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    userChunks := c.Chunker.ChunkMessage(userMsg, messageIndex, chatID, agentName)
    responseChunks := c.Chunker.ChunkMessage(assistantMsg, messageIndex+1, chatID, agentName)

    for _, chunk := range userChunks {
        if err := c.LetheClient.Store(ctx, c.Options.Scope, chunk.Tags, chunk.Content); err != nil {
            slog.Error("LETHE chunk store failed", "error", err)
        }
    }
    for _, chunk := range responseChunks {
        if err := c.LetheClient.Store(ctx, c.Options.Scope, chunk.Tags, chunk.Content); err != nil {
            slog.Error("LETHE chunk store failed", "error", err)
        }
    }
}

// CheckConsent verifies rawText consent for the scope.
func (c *Curator) CheckConsent(ctx context.Context) bool {
    scopes, err := c.LetheClient.Scopes(ctx)
    if err != nil {
        slog.Warn("LETHE scopes call failed", "error", err)
        return false
    }
    for _, sc := range scopes {
        if sc.Scope == c.Options.Scope && sc.RawText {
            return true
        }
    }
    return false
}

func (c *Curator) buildSummaryPrompt(messages []api.Message, summaryType string) string {
    var sb strings.Builder
    switch summaryType {
    case "persona":
        sb.WriteString("Summarize who the AI colleague is based on this conversation. Include their name, values, mistakes, and lessons learned.\n\n")
    case "events":
        sb.WriteString("Summarize the important events that occurred in this conversation, in chronological order.\n\n")
    case "interests":
        sb.WriteString("Summarize the topics the AI colleague is interested in, detected through the conversation.\n\n")
    case "topics":
        sb.WriteString("Summarize the theme of the conversation and the current topic of discussion.\n\n")
    case "important":
        sb.WriteString("Summarize anything the user signalled as important or asked to remember.\n\n")
    }
    for _, msg := range messages {
        sb.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }
    return sb.String()
}

func (c *Curator) callModel(ctx context.Context, model string, prompt string) (string, error) {
    // Uses c.Client.Chat to make a single-turn call
    // Returns the assistant message content
    // ... implementation using c.Client.Chat ...
    return "", nil
}
```

### 4.2 LetheMCPClient (agent/lethe_mcp_client.go)

```go
package agent

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "os/exec"
    "strings"
    "sync"
)

type LetheMCPClient struct {
    cmd     *exec.Cmd
    stdin   io.WriteCloser
    stdout  *bufio.Reader
    mu      sync.Mutex
    nextID  int
}

// NewLetheMCPClient spawns LETHE. Expands ~ via os.UserHomeDir().
func NewLetheMCPClient(lethePath string) (*LetheMCPClient, error) {
    if strings.HasPrefix(lethePath, "~/") {
        home, err := os.UserHomeDir()
        if err != nil {
            return nil, fmt.Errorf("get home dir: %w", err)
        }
        lethePath = filepath.Join(home, lethePath[2:])
    }
    cmd := exec.Command(lethePath)
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    c := &LetheMCPClient{
        cmd:    cmd,
        stdin:  stdin,
        stdout: bufio.NewReader(stdout),
        nextID: 1,
    }
    if err := c.handshake(); err != nil {
        c.Close()
        return nil, err
    }
    return c, nil
}

// handshake sends initialize (gets reply) then notifications/initialized (NO reply).
func (c *LetheMCPClient) handshake() error {
    initReq := map[string]any{
        "jsonrpc": "2.0", "id": c.nextID, "method": "initialize",
        "params": map[string]any{
            "protocolVersion": "2024-11-05",
            "capabilities":   map[string]any{},
            "clientInfo":      map[string]any{"name": "ollama-curator", "version": "1.0"},
        },
    }
    c.nextID++
    if err := c.send(initReq); err != nil {
        return err
    }
    if _, err := c.readRawLine(); err != nil {
        return fmt.Errorf("handshake initialize: %w", err)
    }
    // notifications/initialized — NO reply expected
    notif := map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}
    if err := c.send(notif); err != nil {
        return fmt.Errorf("handshake initialized: %w", err)
    }
    return nil
}

func (c *LetheMCPClient) Store(ctx context.Context, scope string, tags []string, content string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    req := c.toolCall("store", map[string]any{"scope": scope, "content": content, "tags": tags})
    if err := c.send(req); err != nil {
        return err
    }
    return c.checkToolResult()
}

// Recall unwraps result.content[0].text before parsing hits.
func (c *LetheMCPClient) Recall(ctx context.Context, scope string, query string, limit int) ([]LetheHit, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    req := c.toolCall("recall", map[string]any{"scope": scope, "query": query, "limit": limit})
    if err := c.send(req); err != nil {
        return nil, err
    }
    line, err := c.readRawLine()
    if err != nil {
        return nil, err
    }
    var envelope struct {
        Result *struct {
            Content []struct {
                Type string `json:"type"`
                Text string `json:"text"`
            } `json:"content"`
            IsError bool `json:"isError"`
        } `json:"result"`
        Error *struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }
    if err := json.Unmarshal(line, &envelope); err != nil {
        return nil, fmt.Errorf("parse JSON-RPC envelope: %w", err)
    }
    if envelope.Error != nil {
        return nil, fmt.Errorf("JSON-RPC error %d: %s", envelope.Error.Code, envelope.Error.Message)
    }
    if envelope.Result == nil {
        return nil, fmt.Errorf("missing result field")
    }
    if envelope.Result.IsError {
        msg := "unknown"
        if len(envelope.Result.Content) > 0 {
            msg = envelope.Result.Content[0].Text
        }
        return nil, fmt.Errorf("LETHE tool error: %s", msg)
    }
    if len(envelope.Result.Content) == 0 {
        return nil, nil
    }
    toolOutput := envelope.Result.Content[0].Text
    if toolOutput == "" {
        return nil, nil
    }
    var toolResult struct {
        Hits []LetheHit `json:"hits"`
    }
    if err := json.Unmarshal([]byte(toolOutput), &toolResult); err != nil {
        return nil, fmt.Errorf("parse tool output: %w", err)
    }
    return toolResult.Hits, nil
}

// Scopes calls the "scopes" tool. Parses {"scopes":[...]} not [...].
func (c *LetheMCPClient) Scopes(ctx context.Context) ([]ScopeInfo, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    req := c.toolCall("scopes", map[string]any{})
    if err := c.send(req); err != nil {
        return nil, err
    }
    line, err := c.readRawLine()
    if err != nil {
        return nil, err
    }
    var envelope struct {
        Result *struct {
            Content []struct {
                Text string `json:"text"`
            } `json:"content"`
            IsError bool `json:"isError"`
        } `json:"result"`
        Error *struct {
            Message string `json:"message"`
        } `json:"error"`
    }
    if err := json.Unmarshal(line, &envelope); err != nil {
        return nil, err
    }
    if envelope.Error != nil {
        return nil, fmt.Errorf("JSON-RPC error: %s", envelope.Error.Message)
    }
    if envelope.Result == nil || len(envelope.Result.Content) == 0 {
        return nil, nil
    }
    if envelope.Result.IsError {
        return nil, fmt.Errorf("scopes error: %s", envelope.Result.Content[0].Text)
    }
    // Parse {"scopes":[...]} — NOT [...]
    var toolResult struct {
        Scopes []ScopeInfo `json:"scopes"`
    }
    if err := json.Unmarshal([]byte(envelope.Result.Content[0].Text), &toolResult); err != nil {
        return nil, fmt.Errorf("parse scopes: %w", err)
    }
    return toolResult.Scopes, nil
}

func (c *LetheMCPClient) checkToolResult() error {
    line, err := c.readRawLine()
    if err != nil {
        return err
    }
    var envelope struct {
        Result *struct {
            IsError bool `json:"isError"`
            Content []struct {
                Text string `json:"text"`
            } `json:"content"`
        } `json:"result"`
        Error *struct {
            Message string `json:"message"`
        } `json:"error"`
    }
    if err := json.Unmarshal(line, &envelope); err != nil {
        return fmt.Errorf("parse store response: %w", err)
    }
    if envelope.Error != nil {
        return fmt.Errorf("JSON-RPC error: %s", envelope.Error.Message)
    }
    if envelope.Result != nil && envelope.Result.IsError {
        msg := "unknown"
        if len(envelope.Result.Content) > 0 {
            msg = envelope.Result.Content[0].Text
        }
        return fmt.Errorf("LETHE store error: %s", msg)
    }
    return nil
}

func (c *LetheMCPClient) toolCall(toolName string, arguments map[string]any) map[string]any {
    req := map[string]any{
        "jsonrpc": "2.0", "id": c.nextID, "method": "tools/call",
        "params": map[string]any{"name": toolName, "arguments": arguments},
    }
    c.nextID++
    return req
}

func (c *LetheMCPClient) send(req map[string]any) error {
    data, err := json.Marshal(req)
    if err != nil {
        return err
    }
    data = append(data, '\n')
    _, err = c.stdin.Write(data)
    return err
}

func (c *LetheMCPClient) readRawLine() ([]byte, error) {
    line, err := c.stdout.ReadBytes('\n')
    if err != nil {
        return nil, err
    }
    return line, nil
}

func (c *LetheMCPClient) Close() error {
    if c.cmd != nil && c.cmd.Process != nil {
        return c.cmd.Process.Kill()
    }
    return nil
}

// LetheHit matches LETHE's actual recall hit shape.
// Source: MemoryTools.swift L502-518
type LetheHit struct {
    NodeLabels       []string `json:"nodeLabels"`
    NodeDescriptions []string `json:"nodeDescriptions"`
    Category         string   `json:"category"`
    Confidence       float64  `json:"confidence"`
    BM25Norm         float64  `json:"bm25Norm"`
    Cosine           float64  `json:"cosine"`
    Hybrid           float64  `json:"hybrid"`
    Summary          string   `json:"summary"`
    Content          string   `json:"content"` // Only when rawText granted
}

type ScopeInfo struct {
    Scope   string `json:"scope"`
    Mode    string `json:"mode"`
    RawText bool   `json:"rawText"`
}
```

### 4.3 Chunker, Synthesizer, helpers

Same as v9 (confirmed correct by reviewer). Key types:

```go
// Chunker (agent/chunker.go) — splits messages into semantic chunks
type Chunker struct { MaxChunkSize int }
type Chunk struct { MessageIndex, ChunkIndex int; Content string; Tags []string }

// Synthesizer (agent/synthesizer.go) — calls model to synthesize
type Synthesizer struct { Client ChatClient }

// escapeBoundaryTags — replaces 4 exact tag strings (3-arg ReplaceAll)
func escapeBoundaryTags(content string) string

// findLastUserMessage — skips <memory_context> and <relevant_memories> blocks
func findLastUserMessage(messages []api.Message) string

// sliceRecentTurns — turn-based slicing preserving tool call/result pairs
func sliceRecentTurns(messages []api.Message, keepTurns int) []api.Message

// stripMemoryBlocks — removes memory block user messages from final list
// Called in finishRun AFTER compaction, so it works regardless of what
// compaction did to st.messages. No index slicing — no panic risk.
func stripMemoryBlocks(messages []api.Message) []api.Message
```

---

## 5. Session integration

### 5.1 Modified Session struct (agent/session.go)

```go
type Session struct {
    // ... existing fields ...
    Compactor Compactor  // Existing — unchanged, handles post-model overflow
    Curator   *Curator   // New — optional, handles pre-model curation
}

type RunOptions struct {
    // ... existing fields ...
    AgentName string  // NEW — for Curator agent isolation
}
```

### 5.2 buildRunMessages changes (agent/session.go L223-240)

Curation BEFORE preflight budget check:

```go
func (s *Session) buildRunMessages(ctx context.Context, runID string, opts RunOptions) ([]api.Message, error) {
    messages := make([]api.Message, 0, len(opts.Messages)+len(opts.NewMessages))
    for _, msg := range opts.Messages {
        messages = append(messages, sanitizeMessageForRun(msg))
    }
    for _, msg := range opts.NewMessages {
        msg = sanitizeMessageForRun(msg)
        messages = append(messages, msg)
    }

    // Curate BEFORE preflight budget check
    if s.Curator != nil && opts.AgentName != "" {
        curated, err := s.Curator.CurateForRun(ctx, messages, opts.AgentName, opts.ChatID, opts.Model)
        if err != nil {
            s.emit(newErrorEvent(newEventMetadata(runID, opts),
                fmt.Sprintf("curation failed, using full history: %v", err)))
        } else {
            messages = curated
        }
    }

    if err := s.checkPreflightPromptBudget(opts, messages); err != nil {
        s.emit(newErrorEvent(newEventMetadata(runID, opts), err.Error()))
        return nil, err
    }
    return messages, nil
}
```

### 5.3 finishRun changes (agent/session.go L371-384)

StorePostResponse call + stripMemoryBlocks:

```go
func (s *Session) finishRun(ctx context.Context, st *runState) (*RunResult, error) {
    // ... existing finish logic ...

    // Post-response storage
    if s.Curator != nil && st.opts.AgentName != "" {
        var lastUser, lastAssistant api.Message
        for i := len(st.messages) - 1; i >= 0; i-- {
            if st.messages[i].Role == "assistant" && lastAssistant.Content == "" {
                lastAssistant = st.messages[i]
            }
            if st.messages[i].Role == "user" && lastUser.Content == "" {
                if !strings.HasPrefix(st.messages[i].Content, "<memory_context>") &&
                   !strings.HasPrefix(st.messages[i].Content, "<relevant_memories>") {
                    lastUser = st.messages[i]
                    break
                }
            }
        }
        if lastUser.Content != "" && lastAssistant.Content != "" {
            s.Curator.StorePostResponse(ctx, lastUser, lastAssistant,
                len(st.messages), st.opts.AgentName, st.opts.ChatID)
        }
        if s.Curator.shouldGenerateSummaries(len(st.messages)) {
            if err := s.Curator.generateSummaries(ctx, st.messages,
                st.opts.AgentName, st.opts.ChatID, st.opts.Model); err != nil {
                slog.Error("curator summary generation failed", "error", err)
            }
        }
    }

    // Strip memory blocks from returned messages
    cleanMessages := stripMemoryBlocks(st.messages)

    return &RunResult{
        Messages:   cleanMessages,
        Latest:     st.latest,
        WorkingDir: s.WorkingDir,
    }, nil
}
```

---

## 6. TUI activation

### 6.1 cmd/agent_tui.go changes

```go
type agentTUIOptions struct {
    // ... existing fields ...
    Curator    *coreagent.Curator  // NEW
    AgentName  string              // NEW
}

// In GenerateAgentTUI:
import (
    "log/slog"
    "os"
    "path/filepath"
    "strings"
    "github.com/ollama/ollama/cmd/config"
    coreagent "github.com/ollama/ollama/agent"
)

ctxMgmtSettings, err := config.LoadContextManagementSettings(
    filepath.Join(os.Getenv("HOME"), ".ollama", "context_management.json"))
if err != nil {
    slog.Warn("context management settings load failed, using defaults", "error", err)
    ctxMgmtSettings = config.DefaultContextManagementSettings()
}

if ctxMgmtSettings.Mode == "curator" && ctxMgmtSettings.AgentName != "" {
    lethePath := ctxMgmtSettings.LethePath
    if strings.HasPrefix(lethePath, "~/") {
        home, _ := os.UserHomeDir()
        lethePath = filepath.Join(home, lethePath[2:])
    }
    letheClient, err := coreagent.NewLetheMCPClient(lethePath)
    if err != nil {
        slog.Warn("LETHE MCP client failed, falling back to compactor", "error", err)
    } else {
        curator := &coreagent.Curator{
            Client:      client,
            LetheClient: letheClient,
            Chunker:     &coreagent.Chunker{MaxChunkSize: 500},
            Synthesizer: &coreagent.Synthesizer{Client: client},
            Options:      config.ApplySettings(ctxMgmtSettings),
        }
        if !curator.CheckConsent(cmd.Context()) {
            slog.Warn("rawText consent not granted, falling back to compactor",
                "scope", ctxMgmtSettings.Scope)
            letheClient.Close()
        } else {
            opts.Curator = curator
            opts.AgentName = ctxMgmtSettings.AgentName
        }
    }
}

// In the agentchat.Options literal (L114-177), add:
//   Curator:   opts.Curator,
//   AgentName: opts.AgentName,
```

### 6.2 cmd/tui/chat/chat.go changes

```go
type Options struct {
    // ... existing fields ...
    Compactor   coreagent.Compactor
    Curator     *coreagent.Curator  // NEW
    AgentName   string               // NEW
}

// In Session construction (L1146-1156):
session := &coreagent.Session{
    // ... existing fields ...
    Compactor: m.opts.Compactor,
    Curator:   m.opts.Curator,  // NEW
}

// In RunOptions (L1157-1168):
opts := coreagent.RunOptions{
    // ... existing fields ...
    AgentName: m.opts.AgentName,  // NEW
}
```

### 6.3 cmd/config/context_management.go

```go
package config

type ContextManagementSettings struct {
    Mode                 string `json:"mode"`
    AgentName            string `json:"agent_name"`
    SummaryInterval      int    `json:"summary_interval"`
    KeepRecentTurns      int    `json:"keep_recent_turns"`
    KeepRelevantChunks   int    `json:"keep_relevant_chunks"`
    SynthesizeNarrative  bool   `json:"synthesize_narrative"`
    StoreMessagesInLethe bool   `json:"store_in_lethe"`
    Scope               string `json:"scope"`
    LethePath           string `json:"lethe_path"`
}

func DefaultContextManagementSettings() ContextManagementSettings {
    return ContextManagementSettings{
        Mode: "compactor", SummaryInterval: 10, KeepRecentTurns: 5,
        KeepRelevantChunks: 7, SynthesizeNarrative: true,
        StoreMessagesInLethe: true, Scope: "connected:ollama",
        LethePath: "~/.local/bin/lethe-memory-mcp",
    }
}

func LoadContextManagementSettings(path string) (ContextManagementSettings, error) {
    // ... file read and JSON unmarshal ...
}

// ApplySettings creates CuratorOptions from settings.
// In package config to avoid cross-package import issues.
func ApplySettings(s ContextManagementSettings) agent.CuratorOptions {
    return agent.CuratorOptions{
        SummaryInterval:      s.SummaryInterval,
        KeepRecentTurns:      s.KeepRecentTurns,
        KeepRelevantChunks:   s.KeepRelevantChunks,
        SynthesizeNarrative:  s.SynthesizeNarrative,
        StoreMessagesInLethe: s.StoreMessagesInLethe,
        Scope:                s.Scope,
    }
}
```

Note: `ApplySettings` in package `config` referencing `agent.CuratorOptions` requires `config` to import `agent`. If this creates a circular import, move `CuratorOptions` to a shared package or define it in `config` and have `agent` import it. The key requirement is that loaded settings become the values the Curator uses.

---

## 7. Error policy

| Path | Error handling |
|---|---|
| CurateForRun (pre-model) | Return error, caller falls back to full history |
| StorePostResponse (post-response) | Log and continue |
| generateSummaries (post-response) | Log and continue |
| Recall (pre-model) | Return error, caller falls back |
| CheckConsent (startup) | Return false, TUI falls back to compactor |

---

## 8. Testing

### 8.1 Key tests

| Test | What it verifies |
|---|---|
| `TestCuratorFirstMessage` | First message: no summaries, no chunks |
| `TestCuratorSubsequentMessage` | Subsequent: retrieves summaries and chunks |
| `TestCuratorDefaultChatID` | v10 fix: CurateForRun works with empty chatID (uses "default") |
| `TestCuratorStorePostResponse` | StorePostResponse called from finishRun |
| `TestStripMemoryBlocks` | Memory blocks removed from final messages |
| `TestStripMemoryBlocksAfterCompaction` | Works after compaction replaces st.messages |
| `TestCheckConsentReal` | CheckConsent calls scopes and checks rawText |
| `TestCheckConsentRefuses` | Returns false when rawText not granted |
| `TestLetheMCPClientScopes` | Scopes parses {"scopes":[...]} correctly |
| `TestLetheMCPClientHandshake` | Initialize gets reply, initialized gets none |
| `TestLetheMCPClientParsesResultEnvelope` | Recall unwraps result.content[0].text |
| `TestTildeExpansion` | NewLetheMCPClient expands ~ |
| `TestApplySettings` | ApplySettings creates CuratorOptions from settings |
| `TestCuratorChunking` | Chunker called and produces chunks |
| `TestCuratorSynthesis` | Synthesizer calls the model |
| `TestEscapeBoundaryTags` | Boundary tags escaped |
| `TestFindLastUserMessageSkipsMemoryBlocks` | Memory blocks skipped |
| `TestSliceRecentTurnsPreservesToolPairs` | Tool call/result pairs not split |
| `TestCuratorPreModelCuration` | Curation before preflight budget check |
| `TestCuratorProductionWiring` | TUI sets Curator and AgentName on Session |
| `TestFormatSummariesEmpty` | Returns empty when no summaries |

### 8.2 Falsifiable tests

| Test | What a facade would fail |
|---|---|
| `TestCuratorRejectsNoAgentName` | Curator returns nil when AgentName empty |
| `TestCuratorPreservesRecentMessages` | Recent messages included even if LETHE fails |
| `TestCuratorDoesNotGrowIndefinitely` | Context packet does not grow across runs |
| `TestPersistMessagesExcludeMemoryBlocks` | Persisted messages contain no memory blocks |
| `TestStorePostResponseCalledFromFinishRun` | StorePostResponse called in finishRun |
| `TestCheckConsentFailsWhenRawTextFalse` | CheckConsent returns false when rawText=false |
| `TestLetheMCPClientFailsAgainstWrongPeer` | Client fails against non-LETHE peer |
| `TestCurationBeforePreflight` | Curation before checkPreflightPromptBudget |
| `TestCuratorDefaultChatIDWorks` | CurateForRun works when chatID is empty |

---

## 9. Compatibility

- **Existing Compactor**: Unchanged. Handles post-model tool-overflow recovery.
- **MCP tools**: Curator uses direct MCP stdio client, separate from model-driven MCP.
- **Server**: Unchanged. Stateless.
- **API**: Unchanged.
- **Models**: Unchanged.
- **AGORA**: Out of scope.

---

## 10. Open questions

1. Chunking strategy: sentence, paragraph, or topic level?
2. Embedding model: model tokenizer or separate embedding model?
3. Synthesis model: same model or smaller/faster model?
4. rawText consent: auto-grant or manual setup?
5. LETHE client transport: spawn new or connect to existing?
6. Summary update frequency: every 10 messages or topic change?
7. ApplySettings cross-package import: config importing agent, or shared package?

---

## 11. What is NOT in this blueprint

AGORA dual interface — separate future blueprint.

---

*End of document.*