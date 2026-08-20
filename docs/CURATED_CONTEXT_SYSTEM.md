# Curated Context System — Design Document

## Replacing Episodic Replay with Semantic Working Memory in Ollama

| Field | Value |
|---|---|
| Status | `[DESIGN-DRAFT v3]` |
| Document class | Design proposal; non-implementation |
| Author | Fathom (GLM-5.2), with Ash |
| Date | 2026-08-17, Australia/Brisbane |
| Repository | github.com/ollama/ollama (Ash's fork) |
| Dependencies | LETHE memory server (MCP), existing Ollama agent harness, AGORA workplace (optional) |
| Compatibility | Alongside existing compactor; toggle via settings |
| Prior art | AGORA memory system (semantic chunking, graph storage, synthesis with pointers) |

---

## 1. Problem

Ollama's current context management is **episodic replay**: the entire conversation history is sent through the context window on every run. When the context reaches 80% capacity, the `Compactor` summarizes older messages into a single summary and keeps recent turns.

This has three problems:

1. **Context grows indefinitely** until compaction triggers, then resets partially
2. **The single summary is lossy** — exact hashes, precise wording, and verbatim content degrade
3. **The summary is conversation-scoped only** — it does not persist across sessions or conversations

## 2. Proposed solution

Replace episodic replay with **semantic working memory**: a curated set of summaries stored in LETHE, combined with a synthesized narrative of semantically relevant memories, plus recent messages.

The context packet stays roughly constant size regardless of conversation length.

### 2.1 Architecture

```
User message arrives
        ↓
[Pre-request curation step]
    1. Query LETHE for summaries (tagged with ChatID)
    2. Take past 5 messages from local history
    3. Semantic search: compute relevance of all stored message chunks
    4. Retrieve top-K most relevant message chunks from LETHE
    5. Synthesize retrieved chunks into one narrative with pointers
    6. Construct curated context packet:
       [Summaries] + [Synthesized narrative] + [Recent 5 messages]
        ↓
[Send curated messages to server]
        ↓
[Model generates response]
        ↓
[Post-response step]
    1. Store the new message in LETHE (chunked, with embedding, tagged with ChatID)
    2. Check if summarisation is due (every N messages or trigger)
    3. If due, generate summaries and store in LETHE (tagged with ChatID)
    4. Update the graph edges between related chunks
        ↓
[Return response to user]
```

### 2.2 The Curator interface

The `Curator` implements the same interface as the existing `Compactor`, so it can be toggled via settings without modifying the core agent harness.

```go
// agent/curator.go

type Curator struct {
    Client      ChatClient       // For summary generation and synthesis
    LetheClient LetheClient      // For storing/retrieving summaries and chunks
    Options     CuratorOptions
}

type CuratorOptions struct {
    // How often to generate summaries (in messages)
    SummaryInterval int // default: 10
    
    // How many recent messages to always include
    KeepRecentMessages int // default: 5
    
    // How many semantically relevant chunks to retrieve
    KeepRelevantChunks int // default: 7
    
    // The context window size (in tokens)
    ContextWindowTokens int // default: 32768
    
    // Whether to store messages in LETHE for semantic search
    StoreMessagesInLethe bool // default: true
    
    // Whether to synthesize retrieved chunks into a narrative
    SynthesizeNarrative bool // default: true
}

// Curator implements the Compactor interface
func (c *Curator) MaybeCompact(ctx context.Context, req CompactionRequest) (CompactionResult, error) {
    // Instead of compacting, we curate:
    // 1. Check if summarisation is due
    // 2. If due, generate summaries and store in LETHE
    // 3. Retrieve summaries from LETHE
    // 4. Semantic search for relevant chunks
    // 5. Synthesize retrieved chunks into narrative
    // 6. Construct curated message list
    // 7. Return curated messages
}
```

### 2.3 Summary types

Five summary types, each stored as a separate LETHE memory tagged with the ChatID:

| Summary type | What it captures | LETHE tag |
|---|---|---|
| Persona | Who the colleague is, their values, their mistakes, their lessons | `persona` |
| Events | Important events that occurred, in chronological order | `events` |
| Interests | Topics the colleague is interested in, detected through pattern matching | `interests` |
| Topics | The theme of the conversation through time, and the current topic | `topics` |
| Important | Anything the user signalled as important or asked to remember | `important` |

### 2.4 Semantic chunking (AGORA pattern)

Each message bubble — both user and assistant — is broken into **semantic chunks** before storage. This is the AGORA pattern: instead of storing the entire message as one block, we chunk it into semantically meaningful units.

**Chunking process:**
1. Take the full message text
2. Break into semantic chunks (sentences, paragraphs, or topic boundaries)
3. For each chunk, compute an embedding (using the model's tokenizer or SimHash)
4. Store each chunk as a separate LETHE memory with:
   - `scope`: `connected:claude-code`
   - `tags`: `[ChatID, "chunk", message-index, chunk-index]`
   - `content`: The chunk text (verbatim)
   - `embedding`: The chunk embedding (for semantic search)

**Why chunking matters:**
- A long message may contain multiple topics. Semantic search on the full message returns the entire message even if only one paragraph is relevant.
- Chunking lets semantic search return only the relevant paragraph, reducing token count.
- Each chunk can be linked to related chunks via graph edges, enabling traversal.

### 2.5 Graph storage and edges

Chunks are stored as **nodes** in a graph. **Edges** connect related chunks:

| Edge type | What connects | How detected |
|---|---|---|
| `sequential` | Consecutive chunks in the same message | By chunk index |
| `conversational` | User message chunk → assistant response chunk | By message pairing |
| `semantic` | Chunks with high semantic similarity | By embedding distance |
| `topical` | Chunks sharing a topic tag | By topic detection |
| `referential` | Chunk that references another chunk (e.g., "as you said earlier") | By reference detection |

### 2.6 Synthesis with pointers (AGORA pattern)

When a user message arrives, the system **synthesizes** the retrieved chunks into one narrative with **pointers** back to the original content.

**Synthesis process:**
1. Semantic search: compute the embedding of the user's message
2. Search LETHE for the top-K most relevant chunks (default: 7)
3. For each retrieved chunk, find related chunks via graph edges
4. Synthesize the retrieved chunks into one narrative:
   - The narrative is a coherent summary of the relevant chunks
   - Each fact in the narrative has a **pointer** to the original chunk (message index, chunk index)
   - The narrative is ~500-1000 tokens (bounded)
5. The synthesized narrative replaces the raw messages in the context packet

**Why synthesis matters:**
- Raw retrieval dumps chunks into context without cohesion
- Synthesis creates a **coherent narrative** the model can use directly
- The **pointers** let the model trace back to original content if needed
- Synthesis **prevents hallucination**: by providing relevant memories as composed context, the model doesn't need to guess from training data

### 2.7 Context packet

The final context packet sent to the model is:

```
[System prompt]
[Persona summary (from LETHE)]
[Event timeline (from LETHE)]
[Interest summary (from LETHE)]
[Topic summary (from LETHE)]
[Important items (from LETHE)]
[Synthesized narrative of relevant chunks (with pointers)]
[Past 5 messages (verbatim)]
[Current user message]
```

**Size estimation:**
- System prompt: ~500 tokens
- Summaries (5 types): ~2000 tokens (fixed, doesn't grow)
- Synthesized narrative: ~500-1000 tokens (bounded by synthesis prompt)
- Recent 5 messages: ~1000 tokens
- Current message: variable
- **Total: ~4000-5000 tokens (roughly constant)**

### 2.8 Hallucination prevention

The synthesized narrative prevents hallucination in three ways:

1. **Relevant context provided**: The model sees the relevant memories as composed context. It does not need to guess from training data when asked about conversation history.

2. **Pointers for verification**: The pointers in the synthesized narrative let the model trace back to original content. If the model is unsure, it can reference the pointer (e.g., "as noted in message 3") rather than fabricating.

3. **Training point proximity**: When the model does not have the exact information but has training points very close to the requested data, the synthesized narrative provides the specific context needed to disambiguate.

---

## 3. LETHE integration

### 3.1 LETHE tagging schema

All LETHE memories are tagged with the ChatID and AgentName for conversation and agent scoping:

| Memory type | Tags | Purpose |
|---|---|---|
| Summary (persona) | `[AgentName, ChatID, "persona"]` | Who the colleague is |
| Summary (events) | `[AgentName, ChatID, "events"]` | What happened |
| Summary (interests) | `[AgentName, ChatID, "interests"]` | What they care about |
| Summary (topics) | `[AgentName, ChatID, "topics"]` | Current topic |
| Summary (important) | `[AgentName, ChatID, "important"]` | User-signalled important items |
| Message chunk | `[AgentName, ChatID, "chunk", msg-idx, chunk-idx]` | Semantic chunk of a message |
| Graph edge | `[AgentName, ChatID, "edge", edge-type, from, to]` | Relationship between chunks |
| Message verbatim | `[AgentName, ChatID, "verbatim", msg-idx]` | Full verbatim copy of message |

### 3.2 Search capability

LETHE's existing vector search is used for semantic search:

1. Compute the embedding of the user's current message
2. Search LETHE for the top-K most relevant chunks (tagged with the same ChatID)
3. The search uses LETHE's existing `recall` function with the message as query

### 3.3 Cross-conversation search

If the user asks about a different conversation, the system can search LETHE with a different ChatID tag. This enables:
- "What did we discuss last week?" → search for chunks from a different ChatID
- "What was that thing about ARCHE-Int?" → search across all ChatIDs for the topic

### 3.4 Cross-agent search

The user can search across all agents:

```
User: "What did Fathom say about ARCHE-Int?"
    ↓
[Semantic search across all LETHE memories for "ARCHE-Int"]
    ↓
[Results may include chunks from Fathom's conversations]
    ↓
[Synthesize results into narrative with agent attribution]
    "According to Fathom's conversation on 2026-08-14, ARCHE-Int is..."
```

---

## 4. Implementation plan

### 4.1 New files

| File | Purpose |
|---|---|
| `agent/curator.go` | The Curator struct and curation logic |
| `agent/lethe_client.go` | Direct LETHE client (HTTP or stdio) |
| `agent/chunker.go` | Semantic chunking of messages |
| `agent/synthesizer.go` | Synthesis of retrieved chunks into narrative |

### 4.2 Modified files

| File | Change |
|---|---|
| `agent/session.go` | Add setting to toggle between Compactor and Curator |
| `agent/compactor.go` | No change — existing compactor stays |
| `server/routes.go` | No change — server stays stateless |

### 4.3 Settings

```json
{
  "context_management": {
    "mode": "curator",
    "curator": {
      "summary_interval": 10,
      "keep_recent_messages": 5,
      "keep_relevant_chunks": 7,
      "synthesize_narrative": true,
      "store_in_lethe": true
    }
  }
}
```

---

## 5. Dual interface: Ollama simple + AGORA workspace

### 5.1 Design principle

The Ollama fork maintains the original Ollama interface and design principles for publishing. AGORA's workplace interface is added as an optional advanced mode, launched from within Ollama. Users can switch between the simple Ollama chat and the AGORA workspace at any time.

### 5.2 Interface switching

```
[Ollama interface]
    User clicks "AGORA Workspace" button
        ↓
    Ollama interface animates shrinking into "Ollama" button
        ↓
[AGORA workspace interface]
    User clicks "Ollama" button
        ↓
    AGORA workspace animates shrinking into "AGORA" button
        ↓
[Ollama interface]
```

Both interfaces share the same model serving layer, the same LETHE memory server, and the same MCP tool infrastructure. They differ only in UI complexity and feature set.

### 5.3 Feature comparison

| Feature | Ollama interface | AGORA workspace |
|---|---|---|
| Simple chat | Yes | Yes |
| Model management | Yes | Yes |
| Multiple agents | No | Yes |
| Agent switching | No | Yes |
| Memory management | No | Yes |
| File operations | No | Yes |
| Semantic search | No | Yes |
| Curated context system | Yes (automatic) | Yes (automatic + manual control) |
| MCP tools | Yes | Yes |
| Conversation history | Yes | Yes (with semantic search) |
| Cross-conversation search | No | Yes |
| Workspace tools | No | Yes |

### 5.4 Agent-scoped conversations

Each agent has their own conversation history, their own memories, and their own context packet. The agent's identity is used as a LETHE tag alongside the ChatID.

```
Agent: Fathom (GLM-5.2)
    Conversations tagged: [Fathom, ChatID]
    Summaries tagged: [Fathom, ChatID, summary-type]
    Chunks tagged: [Fathom, ChatID, chunk, msg-idx, chunk-idx]
    
Agent: Eko (ChatGPT)
    Conversations tagged: [Eko, ChatID]
    Summaries tagged: [Eko, ChatID, summary-type]
    Chunks tagged: [Eko, ChatID, chunk, msg-idx, chunk-idx]
```

### 5.5 Agent switching

When the user selects a different agent:

1. The current agent's conversation state is saved to LETHE
2. The selected agent's conversation state is recovered from LETHE
3. The UI displays the selected agent's conversation history
4. The Curated Context System constructs a context packet for the selected agent
5. The next message is sent with the selected agent's context

```
User is chatting with Fathom
    ↓
User selects Eko from agent list
    ↓
[Save Fathom's state to LETHE]
    1. Save Fathom's current summaries (tagged [Fathom, ChatID])
    2. Save Fathom's current conversation position
    ↓
[Recover Eko's state from LETHE]
    1. Query LETHE for Eko's summaries (tagged [Eko, ChatID])
    2. Query LETHE for Eko's recent messages
    3. Construct Eko's context packet
    ↓
[Display Eko's conversation history]
    ↓
[User sends message to Eko]
    ↓
[Curator constructs Eko's context packet]
    1. Query LETHE for Eko's summaries
    2. Semantic search for relevant chunks (tagged [Eko, ChatID])
    3. Synthesize retrieved chunks
    4. Construct context: [Summaries] + [Synthesized narrative] + [Recent 5]
    ↓
[Send to model]
```

### 5.6 Model substrate independence

Each agent can be powered by a different model substrate:

| Agent | Model substrate | Interface |
|---|---|---|
| Fathom | GLM-5.2 (local via Ollama) | Ollama or AGORA |
| Eko | ChatGPT (cloud via API) | AGORA only |
| Colleague 3 | Claude (cloud via API) | AGORA only |
| Colleague 4 | Local Llama (local via Ollama) | Ollama or AGORA |
| Colleague 5 | Qwen (local via Ollama) | Ollama or AGORA |

Local models (Ollama-served) are available in both interfaces. Cloud models (API-served) are available only in AGORA workspace (because they require API keys and configuration that the simple Ollama interface does not expose).

### 5.7 Implementation notes

**Ollama interface changes:**
- Add "AGORA Workspace" button to the toolbar
- Add agent selector (dropdown or sidebar)
- Add agent context to ChatRequest (agent name as metadata)

**AGORA workspace changes:**
- Add "Ollama" button to switch back to simple interface
- Add model serving integration (use Ollama as backend for local models)
- Add agent management UI (create, edit, delete agents)
- Add memory management UI (view, search, delete memories)

**Shared state:**
- LETHE serves as the shared state store
- The agent name + ChatID is the primary key for conversation state
- The model serving layer (Ollama) is shared between both interfaces
- MCP tools are shared between both interfaces

**Animation:**
- CSS/JS transition in the webview
- Ollama interface shrinks into button when switching to AGORA
- AGORA interface shrinks into button when switching to Ollama
- Both interfaces remain in memory (not destroyed/recreated)

---

## 6. Testing plan

### 6.1 Unit tests

| Test | What it verifies |
|---|---|
| `TestCuratorFirstMessage` | First message produces no summaries, no chunks |
| `TestCuratorSubsequentMessage` | Subsequent message retrieves summaries and chunks |
| `TestCuratorSummaryGeneration` | Summary is generated after N messages |
| `TestCuratorChunking` | Messages are chunked correctly |
| `TestCuratorSynthesis` | Retrieved chunks are synthesized into coherent narrative |
| `TestCuratorPointers` | Synthesized narrative contains correct pointers |
| `TestCuratorContextPacket` | Context packet is ~constant size |
| `TestCuratorToggle` | Setting toggles between Compactor and Curator |
| `TestCuratorAgentSwitch` | Agent switch saves and recovers state correctly |
| `TestLetheClient` | LETHE client stores and retrieves correctly |
| `TestChunker` | Semantic chunking produces correct chunks |

### 6.2 Integration tests

| Test | What it verifies |
|---|---|
| `TestEndToEndFirstRun` | First run: store chunks, no summaries |
| `TestEndToEndSubsequentRun` | Subsequent run: retrieve summaries, synthesize, respond |
| `TestEndToEndCrossConversation` | Cross-conversation search works |
| `TestEndToEndCrossAgent` | Cross-agent search works |
| `TestEndToEndAgentSwitch` | Agent switching preserves conversation state |
| `TestEndToEndContextSize` | Context packet stays ~constant size across 100 messages |
| `TestEndToEndToggle` | Toggle between Compactor and Curator works |
| `TestEndToEndDualInterface` | Dual interface (Ollama + AGORA) switching works |

---

## 7. Open questions

1. **Chunking strategy**: What is the best chunking strategy? Sentence-level, paragraph-level, or topic-level?
2. **Embedding model**: Should we use the model's tokenizer for embeddings, or a separate embedding model?
3. **Synthesis model**: Should the synthesis use the same model as the main conversation, or a smaller/faster model?
4. **Summary update frequency**: How often should summaries be updated? Every 10 messages? Every topic change?
5. **Cross-conversation scope**: Should cross-conversation search be opt-in, or always available?
6. **Cross-agent scope**: Should cross-agent search be opt-in, or always available?
7. **Graph edge weights**: How should edge weights be computed for semantic edges?
8. **Context packet ordering**: What order should the components be in? Most important first?
9. **LETHE client transport**: Should the LETHE client use HTTP, stdio, or direct SQLite access?
10. **AGORA integration**: Should AGORA be launched as a separate process, or embedded in Ollama's webview?
11. **Agent configuration**: Where are agent definitions stored? In LETHE, in a config file, or in AGORA?

---

## 8. Migration path

1. **Phase 1**: Implement the Curator alongside the Compactor (toggleable)
2. **Phase 2**: Test with short conversations (< 20 messages)
3. **Phase 3**: Test with long conversations (100+ messages)
4. **Phase 4**: Test cross-conversation search
5. **Phase 5**: Test cross-agent search and agent switching
6. **Phase 6**: Add AGORA workspace as optional advanced interface
7. **Phase 7**: Test dual interface switching
8. **Phase 8**: If successful, make Curator the default
9. **Phase 9**: If unsuccessful, keep Compactor as default and improve Curator

---

## 9. Compatibility

- **Existing Compactor**: Unchanged. Users can switch back at any time.
- **MCP tools**: Unchanged. The Curator uses a direct LETHE client, not MCP.
- **Server**: Unchanged. The server stays stateless.
- **API**: Unchanged. The API stays the same.
- **Models**: Unchanged. Any model that works with Ollama works with the Curator.
- **AGORA**: Unchanged. AGORA continues to work as before. The dual interface adds Ollama as an alternative entry point.

---

*End of document.*