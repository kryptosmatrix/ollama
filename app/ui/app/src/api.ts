import {
  approvalRequestBody,
  type ApprovalDecision,
} from "@/utils/toolApproval";
import {
  ChatResponse,
  ChatsResponse,
  ChatEvent,
  DownloadEvent,
  ErrorEvent,
  InferenceComputeResponse,
  MCPServer,
  MCPDiscoveredServer,
  MCPRegistryEntry,
  ModelCapabilitiesResponse,
  Model,
  ChatRequest,
  Settings,
  User,
} from "@/gotypes";
import { parseJsonlFromResponse } from "./util/jsonl-parsing";
import { ollamaClient as ollama } from "./lib/ollama-client";
import type { ModelResponse } from "ollama/browser";
import { API_BASE, OLLAMA_DOT_COM } from "./lib/config";

// Extend Model class with utility methods
declare module "@/gotypes" {
  interface Model {
    isCloud(): boolean;
  }
}

Model.prototype.isCloud = function (): boolean {
  return this.model.endsWith("cloud");
};

export type CloudStatusSource = "env" | "config" | "both" | "none";
export interface CloudStatusResponse {
  disabled: boolean;
  source: CloudStatusSource;
}
// Helper function to convert Uint8Array to base64
function uint8ArrayToBase64(uint8Array: Uint8Array): string {
  const chunkSize = 0x8000; // 32KB chunks to avoid stack overflow
  let binary = "";

  for (let i = 0; i < uint8Array.length; i += chunkSize) {
    const chunk = uint8Array.subarray(i, i + chunkSize);
    binary += String.fromCharCode(...chunk);
  }

  return btoa(binary);
}

export async function fetchUser(): Promise<User | null> {
  const response = await fetch(`${API_BASE}/api/me`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (response.ok) {
    const userData: User = await response.json();

    if (userData.avatarurl && !userData.avatarurl.startsWith("http")) {
      userData.avatarurl = `${OLLAMA_DOT_COM}${userData.avatarurl}`;
    }

    return userData;
  }

  if (response.status === 401 || response.status === 403) {
    return null;
  }

  throw new Error(`Failed to fetch user: ${response.status}`);
}

export async function fetchConnectUrl(): Promise<string> {
  const response = await fetch(`${API_BASE}/api/me`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (response.status === 401) {
    const data = await response.json();
    if (data.signin_url) {
      return data.signin_url;
    }
  }

  throw new Error("Failed to fetch connect URL");
}

export async function disconnectUser(): Promise<void> {
  const response = await fetch(`${API_BASE}/api/signout`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!response.ok) {
    throw new Error("Failed to disconnect user");
  }
}

export async function getChats(): Promise<ChatsResponse> {
  const response = await fetch(`${API_BASE}/api/v1/chats`);
  const data = await response.json();
  return new ChatsResponse(data);
}

export async function getChat(chatId: string): Promise<ChatResponse> {
  const response = await fetch(`${API_BASE}/api/v1/chat/${chatId}`);
  const data = await response.json();
  return new ChatResponse(data);
}

export async function getModels(query?: string): Promise<Model[]> {
  try {
    const { models: modelsResponse } = await ollama.list();

    let models: Model[] = modelsResponse
      .filter((m: ModelResponse) => {
        const families = m.details?.families;

        if (!families || families.length === 0) {
          return true;
        }

        const isBertOnly = families.every((family: string) =>
          family.toLowerCase().includes("bert"),
        );

        return !isBertOnly;
      })
      .map((m: ModelResponse) => {
        // Remove the latest tag from the returned model
        const modelName = m.name.replace(/:latest$/, "");

        return new Model({
          model: modelName,
          digest: m.digest,
          modified_at: m.modified_at ? new Date(m.modified_at) : undefined,
        });
      });

    // Filter by query if provided
    if (query) {
      const normalizedQuery = query.toLowerCase().trim();

      const filteredModels = models.filter((m: Model) => {
        return m.model.toLowerCase().startsWith(normalizedQuery);
      });

      let exactMatch = false;
      for (const m of filteredModels) {
        if (m.model.toLowerCase() === normalizedQuery) {
          exactMatch = true;
          break;
        }
      }

      // Add query if it's in the registry and not already in the list
      if (!exactMatch) {
        const result = await getModelUpstreamInfo(new Model({ model: query }));
        const existsUpstream = result.exists;
        if (existsUpstream) {
          filteredModels.push(new Model({ model: query }));
        }
      }

      models = filteredModels;
    }

    return models;
  } catch (err) {
    throw new Error(`Failed to fetch models: ${err}`);
  }
}

export async function getModelCapabilities(
  modelName: string,
): Promise<ModelCapabilitiesResponse> {
  try {
    const showResponse = await ollama.show({ model: modelName });

    return new ModelCapabilitiesResponse({
      capabilities: Array.isArray(showResponse.capabilities)
        ? showResponse.capabilities
        : [],
    });
  } catch (error) {
    // Model might not be downloaded yet, return empty capabilities
    console.error(`Failed to get capabilities for ${modelName}:`, error);
    return new ModelCapabilitiesResponse({ capabilities: [] });
  }
}

export type ChatEventUnion = ChatEvent | DownloadEvent | ErrorEvent;

export async function* sendMessage(
  chatId: string,
  message: string,
  model: Model,
  attachments?: Array<{ filename: string; data: Uint8Array }>,
  signal?: AbortSignal,
  index?: number,
  webSearch?: boolean,
  fileTools?: boolean,
  forceUpdate?: boolean,
  think?: boolean | string,
): AsyncGenerator<ChatEventUnion> {
  // Convert Uint8Array to base64 for JSON serialization
  const serializedAttachments = attachments?.map((att) => ({
    filename: att.filename,
    data: uint8ArrayToBase64(att.data),
  }));

  // Send think parameter when it's explicitly set (true, false, or a non-empty string).
  const shouldSendThink =
    think !== undefined &&
    (typeof think === "boolean" || (typeof think === "string" && think !== ""));

  const response = await fetch(`${API_BASE}/api/v1/chat/${chatId}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(
      new ChatRequest({
        model: model.model,
        prompt: message,
        ...(index !== undefined ? { index } : {}),
        ...(serializedAttachments !== undefined
          ? { attachments: serializedAttachments }
          : {}),
        // Always send web_search as a boolean value (default to false)
        web_search: webSearch ?? false,
        file_tools: fileTools ?? false,
        ...(forceUpdate !== undefined ? { forceUpdate } : {}),
        ...(shouldSendThink ? { think } : {}),
      }),
    ),
    signal,
  });

  for await (const event of parseJsonlFromResponse<ChatEventUnion>(response)) {
    switch (event.eventName) {
      case "download":
        yield new DownloadEvent(event);
        break;
      case "error":
        yield new ErrorEvent(event);
        break;
      default:
        yield new ChatEvent(event);
        break;
    }
  }
}

/**
 * Answers a tool call that is waiting for approval.
 *
 * The waiting call is inside a different, still-open response, so this request
 * is the only way it ever resumes. A 409 means it is no longer waiting — it was
 * already answered, it timed out, or the chat ended — which is reported rather
 * than swallowed, because a button that silently did nothing would leave the
 * user believing they had answered.
 */
export async function respondToToolApproval(
  chatId: string,
  approvalId: string,
  decision: ApprovalDecision,
): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/chat/${chatId}/approval`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(approvalRequestBody(approvalId, decision)),
  });
  if (response.status === 409) {
    throw new Error("That tool call is no longer waiting for an answer.");
  }
  if (!response.ok) {
    throw new Error(`Failed to answer the approval: ${response.status}`);
  }
}

export interface AddMCPServerRequest {
  name: string;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
}

async function mcpRequest(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const response = await fetch(`${API_BASE}/api/v1/mcp${path}`, init);
  if (!response.ok) {
    let detail = "";
    try {
      detail = (await response.text()).trim();
    } catch {
      detail = "";
    }
    throw new Error(detail || `Request failed: ${response.status}`);
  }
  return response;
}

export async function listMCPServers(): Promise<MCPServer[]> {
  const response = await mcpRequest("");
  const data = await response.json();
  return (data.servers ?? []).map((server: unknown) => new MCPServer(server));
}

export async function addMCPServer(
  request: AddMCPServerRequest,
): Promise<void> {
  await mcpRequest("", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}

export async function setMCPServerEnabled(
  name: string,
  enabled: boolean,
): Promise<void> {
  await mcpRequest(`/${encodeURIComponent(name)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled }),
  });
}

/**
 * Approves a server to run.
 *
 * `runs` is the command line the page displayed. The server checks it against
 * what is on disk and refuses if they differ, so a stale page or a
 * configuration edited underneath cannot approve something the user never
 * read.
 */
export async function approveMCPServer(
  name: string,
  runs: string,
): Promise<void> {
  await mcpRequest(`/${encodeURIComponent(name)}/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ runs }),
  });
}

export async function removeMCPServer(name: string): Promise<void> {
  await mcpRequest(`/${encodeURIComponent(name)}`, { method: "DELETE" });
}

/**
 * Starts a browser sign-in for a remote server.
 *
 * This returns as soon as the sign-in has started, not when it finishes: the
 * user is in a browser at that point, and how long they take there is not a
 * request timeout. The outcome shows up in the server's status.
 */
export async function signInMCPServer(name: string): Promise<void> {
  await mcpRequest(`/${encodeURIComponent(name)}/signin`, { method: "POST" });
}

/**
 * Revokes a server's token and deletes it from this machine.
 *
 * Resolves with the server as it now stands. When the token could not be
 * revoked at the server it was still deleted here, and `error` says so — the
 * caller must show it rather than treat the call as a clean sign-out.
 */
export async function signOutMCPServer(name: string): Promise<MCPServer> {
  const response = await mcpRequest(`/${encodeURIComponent(name)}/signout`, {
    method: "POST",
  });
  return new MCPServer(await response.json());
}

export interface MCPRegistryPage {
  entries: MCPRegistryEntry[];
  nextCursor?: string;
  notVetted: boolean;
}

/** Searches the official MCP Registry. An empty query lists everything. */
export async function browseMCPRegistry(
  search: string,
  cursor?: string,
): Promise<MCPRegistryPage> {
  const params = new URLSearchParams();
  if (search.trim() !== "") params.set("search", search.trim());
  if (cursor) params.set("cursor", cursor);

  const response = await fetch(
    `${API_BASE}/api/v1/mcp-registry?${params.toString()}`,
  );
  if (!response.ok) {
    const detail = (await response.text().catch(() => "")).trim();
    throw new Error(
      detail || `Could not reach the registry: ${response.status}`,
    );
  }
  const data = await response.json();
  return {
    entries: (data.entries ?? []).map(
      (entry: unknown) => new MCPRegistryEntry(entry),
    ),
    nextCursor: data.nextCursor,
    notVetted: Boolean(data.notVetted),
  };
}

/**
 * Asks what installing one entry would write, at the moment of the decision.
 *
 * The browse list may be minutes old by the time the user clicks, and what
 * they are shown before agreeing must be current — so this asks again rather
 * than trusting the row.
 */
export async function resolveMCPRegistryEntry(
  name: string,
): Promise<MCPRegistryEntry> {
  const response = await fetch(`${API_BASE}/api/v1/mcp-registry/resolve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (!response.ok) {
    const detail = (await response.text().catch(() => "")).trim();
    throw new Error(
      detail || `Could not resolve the entry: ${response.status}`,
    );
  }
  return new MCPRegistryEntry(await response.json());
}

export interface MCPDiscovery {
  servers: MCPDiscoveredServer[];
  /** Every path that was looked at, existing or not. */
  searched: string[];
  /** A failure that did not stop the rest of the search. */
  error?: string;
}

function readDiscovery(data: unknown): MCPDiscovery {
  const body = (data ?? {}) as {
    servers?: unknown[];
    searched?: string[];
    error?: string;
  };
  return {
    servers: (body.servers ?? []).map(
      (server: unknown) => new MCPDiscoveredServer(server),
    ),
    searched: body.searched ?? [],
    error: body.error,
  };
}

/**
 * Lists MCP servers other applications on this machine are configured with.
 *
 * This reads named files and contacts nothing, which is why it is a plain
 * read while the probe below is not.
 */
export async function discoverMCPServers(): Promise<MCPDiscovery> {
  const response = await fetch(`${API_BASE}/api/v1/mcp-discover`);
  if (!response.ok) {
    const detail = (await response.text().catch(() => "")).trim();
    throw new Error(
      detail || `Could not search this machine: ${response.status}`,
    );
  }
  return readDiscovery(await response.json());
}

/**
 * Looks for MCP servers answering on this machine right now.
 *
 * A POST because it is an act: it contacts every listening loopback port with
 * the MCP handshake. Nothing calls it except a user asking for it.
 */
export async function probeMCPServers(): Promise<MCPDiscovery> {
  const response = await fetch(`${API_BASE}/api/v1/mcp-probe`, {
    method: "POST",
  });
  if (!response.ok) {
    const detail = (await response.text().catch(() => "")).trim();
    throw new Error(
      detail || `Could not check this machine: ${response.status}`,
    );
  }
  return readDiscovery(await response.json());
}

export async function getSettings(): Promise<{
  settings: Settings;
}> {
  const response = await fetch(`${API_BASE}/api/v1/settings`);
  if (!response.ok) {
    throw new Error("Failed to fetch settings");
  }
  const data = await response.json();
  return {
    settings: new Settings(data.settings),
  };
}

export async function updateSettings(settings: Settings): Promise<{
  settings: Settings;
}> {
  const response = await fetch(`${API_BASE}/api/v1/settings`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(settings),
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Failed to update settings");
  }
  const data = await response.json();
  return {
    settings: new Settings(data.settings),
  };
}

export async function updateCloudSetting(
  enabled: boolean,
): Promise<CloudStatusResponse> {
  const response = await fetch(`${API_BASE}/api/v1/cloud`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ enabled }),
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Failed to update cloud setting");
  }

  const data = await response.json();
  return {
    disabled: Boolean(data.disabled),
    source: (data.source as CloudStatusSource) || "none",
  };
}

export async function renameChat(chatId: string, title: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/chat/${chatId}/rename`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title: title.trim() }),
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Failed to rename chat");
  }
}

export async function deleteChat(chatId: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/chat/${chatId}`, {
    method: "DELETE",
  });
  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || "Failed to delete chat");
  }
}

// Get upstream information for model staleness checking
export async function getModelUpstreamInfo(
  model: Model,
): Promise<{ stale: boolean; exists: boolean; error?: string }> {
  try {
    const response = await fetch(`${API_BASE}/api/v1/model/upstream`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        model: model.model,
      }),
    });

    if (!response.ok) {
      console.warn(
        `Failed to check upstream for ${model.model}: ${response.status}`,
      );
      return { stale: false, exists: false };
    }

    const data = await response.json();

    if (data.error) {
      console.warn(`Upstream check: ${data.error}`);
      return { stale: false, exists: false, error: data.error };
    }

    return { stale: !!data.stale, exists: true };
  } catch (error) {
    console.warn(`Error checking model staleness:`, error);
    return { stale: false, exists: false };
  }
}

export async function* pullModel(
  modelName: string,
  signal?: AbortSignal,
): AsyncGenerator<{
  status: string;
  digest?: string;
  total?: number;
  completed?: number;
  done?: boolean;
}> {
  const response = await fetch(`${API_BASE}/api/v1/models/pull`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name: modelName }),
    signal,
  });

  if (!response.ok) {
    throw new Error(`Failed to pull model: ${response.statusText}`);
  }

  for await (const event of parseJsonlFromResponse<{
    status: string;
    digest?: string;
    total?: number;
    completed?: number;
    done?: boolean;
  }>(response)) {
    yield event;
  }
}

export interface ModelRecommendation {
  model: string;
  description: string;
  context_length?: number;
  max_output_tokens?: number;
  vram_bytes?: number;
}

export interface ModelRecommendationsResponse {
  recommendations: ModelRecommendation[];
}

export async function getModelRecommendations(): Promise<
  ModelRecommendation[]
> {
  const response = await fetch(
    `${API_BASE}/api/experimental/model-recommendations`,
  );
  if (!response.ok) {
    throw new Error(
      `Failed to fetch model recommendations: ${response.statusText}`,
    );
  }
  const data: ModelRecommendationsResponse = await response.json();
  return data.recommendations || [];
}

export async function getInferenceCompute(): Promise<InferenceComputeResponse> {
  const response = await fetch(`${API_BASE}/api/v1/inference-compute`);
  if (!response.ok) {
    throw new Error(
      `Failed to fetch inference compute: ${response.statusText}`,
    );
  }

  const data = await response.json();
  return new InferenceComputeResponse(data);
}

export async function fetchHealth(): Promise<boolean> {
  try {
    // Use the /api/version endpoint as a health check
    const response = await fetch(`${API_BASE}/api/version`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (response.ok) {
      const data = await response.json();
      // If we get a version back, the server is healthy
      return !!data.version;
    }

    return false;
  } catch (error) {
    console.error("Error checking health:", error);
    return false;
  }
}

export async function getCloudStatus(): Promise<CloudStatusResponse | null> {
  const response = await fetch(`${API_BASE}/api/v1/cloud`);
  if (!response.ok) {
    throw new Error(`Failed to fetch cloud status: ${response.status}`);
  }

  const data = await response.json();
  return {
    disabled: Boolean(data.disabled),
    source: (data.source as CloudStatusSource) || "none",
  };
}

export type TTSStatus = {
  has_api_key: boolean;
  voice_id: string;
  model_id: string;
  speed: number;
  cache_enabled: boolean;
  cache_clear_pending: boolean;
  secret_store: string;
};

export type TTSVoice = {
  voice_id: string;
  name: string;
  category?: string;
};

async function ttsError(response: Response): Promise<Error> {
  let message = `Speech request failed: ${response.status}`;
  try {
    const body = (await response.json()) as { error?: string };
    if (body.error) {
      message = body.error;
    }
  } catch {
    // keep the status message
  }
  const error = new Error(message) as Error & { status: number };
  error.status = response.status;
  return error;
}

export async function getTTSStatus(): Promise<TTSStatus> {
  const response = await fetch(`${API_BASE}/api/v1/tts/status`);
  if (!response.ok) {
    throw await ttsError(response);
  }
  return response.json();
}

export async function putTTSKey(apiKey: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/tts/key`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ api_key: apiKey }),
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
}

export async function deleteTTSKey(): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/tts/key`, {
    method: "DELETE",
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
}

export async function updateTTSSettings(patch: {
  voice_id?: string;
  model_id?: string;
  speed?: number;
  cache_enabled?: boolean;
}): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/tts/settings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
}

export async function getTTSVoices(): Promise<TTSVoice[]> {
  const response = await fetch(`${API_BASE}/api/v1/tts/voices`);
  if (!response.ok) {
    throw await ttsError(response);
  }
  const data = (await response.json()) as { voices?: TTSVoice[] };
  return data.voices ?? [];
}

export async function clearTTSCache(): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/tts/cache/clear`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "{}",
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
}

export async function commitTTSCache(fingerprint: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/tts/cache/commit`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ fingerprint }),
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
}

export async function speakMessageChunk(
  text: string,
  chunkIndex: number,
  signal?: AbortSignal,
): Promise<{
  blob: Blob;
  contentType: string;
  chunkIndex: number;
  chunkCount: number;
  fingerprint: string;
}> {
  const response = await fetch(`${API_BASE}/api/v1/tts/speak`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text, chunk_index: chunkIndex }),
    signal,
  });
  if (!response.ok) {
    throw await ttsError(response);
  }
  const contentType = response.headers.get("Content-Type") || "";
  const mime = contentType.split(";")[0].trim().toLowerCase();
  if (
    mime !== "audio/mpeg" &&
    mime !== "audio/mp3" &&
    mime !== "application/octet-stream"
  ) {
    throw new Error("ElevenLabs returned data that was not audio.");
  }
  return {
    blob: await response.blob(),
    contentType,
    chunkIndex: Number(response.headers.get("X-Ollama-TTS-Chunk-Index") || chunkIndex),
    chunkCount: Number(response.headers.get("X-Ollama-TTS-Chunk-Count") || 1),
    fingerprint: response.headers.get("X-Ollama-TTS-Fingerprint") || "",
  };
}
