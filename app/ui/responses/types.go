//go:build windows || darwin

package responses

import (
	"time"

	"github.com/ollama/ollama/app/store"
	"github.com/ollama/ollama/types/model"
)

type ChatInfo struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	UserExcerpt string    `json:"userExcerpt"`
	CreatedAt   time.Time `json:"createdAt" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
	UpdatedAt   time.Time `json:"updatedAt" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
}

type ChatsResponse struct {
	ChatInfos []ChatInfo `json:"chatInfos"`
}

type ChatResponse struct {
	Chat store.Chat `json:"chat"`
}

type Model struct {
	Model      string     `json:"model"`
	Digest     string     `json:"digest,omitempty"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

type ModelsResponse struct {
	Models []Model `json:"models"`
}

type InferenceCompute struct {
	Library string `json:"library"`
	Variant string `json:"variant"`
	Compute string `json:"compute"`
	Driver  string `json:"driver"`
	Name    string `json:"name"`
	VRAM    string `json:"vram"`
}

type InferenceComputeResponse struct {
	InferenceComputes    []InferenceCompute `json:"inferenceComputes"`
	DefaultContextLength int                `json:"defaultContextLength"`
}

type ModelCapabilitiesResponse struct {
	Capabilities []model.Capability `json:"capabilities"`
}

// ChatEvent is for regular chat messages and assistant interactions
type ChatEvent struct {
	EventName string `json:"eventName" ts_type:"\"chat\" | \"thinking\" | \"assistant_with_tools\" | \"tool_call\" | \"tool\" | \"tool_result\" | \"tool_approval\" | \"done\" | \"chat_created\""`

	// Chat/Assistant message fields
	Content           *string    `json:"content,omitempty"`
	Thinking          *string    `json:"thinking,omitempty"`
	ThinkingTimeStart *time.Time `json:"thinkingTimeStart,omitempty" ts_type:"Date | undefined" ts_transform:"__VALUE__ && new Date(__VALUE__)"`
	ThinkingTimeEnd   *time.Time `json:"thinkingTimeEnd,omitempty" ts_type:"Date | undefined" ts_transform:"__VALUE__ && new Date(__VALUE__)"`

	// Tool-related fields
	ToolCalls      []store.ToolCall `json:"toolCalls,omitempty"`
	ToolCall       *store.ToolCall  `json:"toolCall,omitempty"`
	ToolName       *string          `json:"toolName,omitempty"`
	ToolResult     *bool            `json:"toolResult,omitempty"`
	ToolResultData any              `json:"toolResultData,omitempty"`

	// Chat creation fields
	ChatID *string `json:"chatId,omitempty"`

	// Approval fields, sent with a "tool_approval" event when a tool may not
	// run until the user answers. The answer is posted back to
	// /api/v1/chat/{id}/approval carrying ApprovalID.
	ApprovalID    *string        `json:"approvalId,omitempty"`
	ApprovalScope *string        `json:"approvalScope,omitempty"`
	ApprovalArgs  map[string]any `json:"approvalArgs,omitempty"`

	// Tool state field from the new code
	ToolState any `json:"toolState,omitempty"`
}

// DownloadEvent is for model download progress
type DownloadEvent struct {
	EventName string `json:"eventName" ts_type:"\"download\""`
	Total     int64  `json:"total" ts_type:"number"`
	Completed int64  `json:"completed" ts_type:"number"`
	Done      bool   `json:"done" ts_type:"boolean"`
}

// ErrorEvent is for error messages
type ErrorEvent struct {
	EventName string `json:"eventName" ts_type:"\"error\""`
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`    // Optional error code for different error types
	Details   string `json:"details,omitempty"` // Optional additional details
}

type SettingsResponse struct {
	Settings store.Settings `json:"settings"`
}

type HealthResponse struct {
	Healthy bool `json:"healthy"`
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Bio       string `json:"bio,omitempty"`
	AvatarURL string `json:"avatarurl,omitempty"`
	FirstName string `json:"firstname,omitempty"`
	LastName  string `json:"lastname,omitempty"`
	Plan      string `json:"plan,omitempty"`
}

type Attachment struct {
	Filename string `json:"filename"`
	Data     string `json:"data,omitempty"` // omitempty = optional, no data = existing file reference
}

type ChatRequest struct {
	Model       string       `json:"model"`
	Prompt      string       `json:"prompt"`
	Index       *int         `json:"index,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	WebSearch   *bool        `json:"web_search,omitempty"`
	FileTools   *bool        `json:"file_tools,omitempty"`
	ForceUpdate bool         `json:"forceUpdate,omitempty"`
	Think       any          `json:"think,omitempty"`
}

type Error struct {
	Error string `json:"error"`
}

type ModelUpstreamResponse struct {
	Stale bool   `json:"stale"`
	Error string `json:"error,omitempty"`
}

// Serializable data for the browser state
type BrowserStateData struct {
	PageStack  []string         `json:"page_stack"`  // Sequential list of page URLs
	ViewTokens int              `json:"view_tokens"` // Number of tokens to show in viewport
	URLToPage  map[string]*Page `json:"url_to_page"` // URL to page contents
}

// Page represents the contents of a page
type Page struct {
	URL       string         `json:"url"`
	Title     string         `json:"title"`
	Text      string         `json:"text"`
	Lines     []string       `json:"lines"`
	Links     map[int]string `json:"links,omitempty" ts_type:"Record<number, string>"`
	FetchedAt time.Time      `json:"fetched_at"`
}

// MCPTool is one tool a connected MCP server is offering the model.
type MCPTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// MCPSkippedTool is a tool Ollama refused to offer, and why. These are shown
// rather than swallowed: a user whose tool is missing needs to know it was
// rejected rather than absent.
type MCPSkippedTool struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// MCPServer is one configured MCP server as the app presents it: what it would
// run, whether Ollama may run it, and what it is currently offering.
type MCPServer struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Transport string `json:"transport"`
	// Runs is the command line or URL, shown verbatim. It is what the user
	// reads before approving.
	Runs    string `json:"runs"`
	Enabled bool   `json:"enabled"`
	// Approved reports whether this exact spec has been agreed to.
	Approved bool `json:"approved"`
	// Changed reports that the server was approved once but has been edited
	// since, so what it would now run has never been agreed to.
	Changed bool `json:"changed,omitempty"`
	// PreviouslyRan is the command line that was approved, shown beside the
	// current one so the change is visible rather than merely detected.
	PreviouslyRan string           `json:"previouslyRan,omitempty"`
	Error         string           `json:"error,omitempty"`
	Tools         []MCPTool        `json:"tools,omitempty"`
	Skipped       []MCPSkippedTool `json:"skipped,omitempty"`
	// CanSignIn reports that this is a remote server, so signing in is
	// something that could apply to it. Whether it actually needs a sign-in is
	// something only the server can say, and it says so with a 401.
	CanSignIn bool `json:"canSignIn,omitempty"`
	// SignedIn reports that a token for this server is stored on this machine.
	SignedIn bool `json:"signedIn,omitempty"`
	// SigningIn reports that a sign-in is in flight, so the browser window the
	// user is looking at belongs to this server.
	SigningIn bool `json:"signingIn,omitempty"`
	// TokenStore says in one line where a token is kept and how well it is
	// protected. It is shown wherever a sign-in is offered: someone signing in
	// to a third-party service is entitled to know where the credential ends
	// up before they create one.
	TokenStore string `json:"tokenStore,omitempty"`
}

type MCPServersResponse struct {
	Servers []MCPServer `json:"servers"`
}

// MCPRegistryEntry is one server from the official MCP Registry, as the browse
// surface presents it.
//
// Runs is the exact command line or URL an install would write, resolved here
// so the user reads the real thing rather than a name. Installable is false
// when Ollama does not know how to run the entry, and Reason says why — an
// entry is never presented with an install button that would produce a command
// line nobody has verified.
type MCPRegistryEntry struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	// Publisher is the namespace half of the reverse-DNS name. It is the only
	// provenance the registry carries.
	Publisher  string `json:"publisher"`
	Repository string `json:"repository,omitempty"`
	WebsiteURL string `json:"websiteUrl,omitempty"`

	Installable bool   `json:"installable"`
	Reason      string `json:"reason,omitempty"`
	Transport   string `json:"transport,omitempty"`
	Runs        string `json:"runs,omitempty"`
	// SuggestedName is the name the server would be configured under. It is
	// derived from the entry, and the user may change it.
	SuggestedName string `json:"suggestedName,omitempty"`

	// The resolved specification, carried structurally so the interface never
	// has to parse Runs back apart. Runs is for the user to read; these are
	// what an install writes. Two representations of one thing would drift,
	// and the one the user agreed to must be the one that is stored.
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	// Variables are the environment references the entry declares, so the user
	// can be told which values they must set before it will work.
	Variables []string `json:"variables,omitempty"`
}

type MCPRegistryResponse struct {
	Entries []MCPRegistryEntry `json:"entries"`
	// NextCursor is empty when there are no more results.
	NextCursor string `json:"nextCursor,omitempty"`
	// NotVetted is always true. The registry is an open-publish metadata
	// service; a listing is a claim by its publisher and not something Ollama
	// has checked. It is a field rather than a comment so the interface cannot
	// quietly stop saying it.
	NotVetted bool `json:"notVetted"`
}
