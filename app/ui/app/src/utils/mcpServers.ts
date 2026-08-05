import type { MCPServer } from "@/gotypes";

/** How a server's state should read to a person, and what they can do about it. */
export interface MCPServerPresentation {
  /** A short phrase for the status pill. */
  label: string;
  /** Longer explanation, when there is something to explain. */
  detail: string;
  /** The server needs approving before Ollama will run it. */
  needsApproval: boolean;
  /** Something is wrong and the user must act; distinct from merely switched off. */
  attention: boolean;
}

/**
 * Describes one server's state for display.
 *
 * The order matters and mirrors the backend's: a broken entry first, then the
 * user's own switch, then approval, then the live connection. A server the user
 * switched off should read as off rather than as awaiting approval, and a
 * server that cannot run as configured should say so before anything else.
 */
export function presentMCPServer(server: MCPServer): MCPServerPresentation {
  if (server.status === "invalid") {
    return {
      label: "Misconfigured",
      detail: server.error ?? "This server cannot run as configured.",
      needsApproval: false,
      attention: true,
    };
  }
  if (!server.enabled) {
    return { label: "Off", detail: "", needsApproval: false, attention: false };
  }
  if (server.changed) {
    return {
      label: "Changed",
      detail: server.previouslyRan
        ? `Previously approved to run: ${server.previouslyRan}`
        : "This server has changed since it was approved.",
      needsApproval: true,
      attention: true,
    };
  }
  if (!server.approved) {
    return {
      label: "Not approved",
      detail: "Ollama will not run this server until you approve it.",
      needsApproval: true,
      attention: true,
    };
  }
  if (server.status === "failed") {
    return {
      label: "Unavailable",
      detail: server.error ?? "This server could not be reached.",
      needsApproval: false,
      attention: true,
    };
  }
  if (server.status === "connected") {
    const count = server.tools?.length ?? 0;
    return {
      label: "Connected",
      detail: `${count} ${count === 1 ? "tool" : "tools"}`,
      needsApproval: false,
      attention: false,
    };
  }
  return {
    label: "Connecting",
    detail: server.error ?? "",
    needsApproval: false,
    attention: false,
  };
}

/**
 * Parses a pasted configuration block into servers to add.
 *
 * Every other MCP client distributes servers as a JSON snippet, so a user
 * should be able to paste one rather than retype it. Both the wrapped form —
 * an object with an "mcpServers" key — and a bare map of servers are accepted,
 * because both appear in the wild.
 *
 * Nothing here approves anything: these become servers the user must still
 * read and agree to.
 */
export function parsePastedServers(text: string): AddMCPServerInput[] {
  const parsed: unknown = JSON.parse(text);
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("That is not an MCP server configuration.");
  }

  const record = parsed as Record<string, unknown>;
  const wrapped = record.mcpServers;
  const servers =
    wrapped && typeof wrapped === "object" && !Array.isArray(wrapped)
      ? (wrapped as Record<string, unknown>)
      : record;

  const names = Object.keys(servers);
  if (names.length === 0) {
    throw new Error("No servers found in that configuration.");
  }

  return names.sort().map((name) => {
    const entry = servers[name];
    if (entry === null || typeof entry !== "object" || Array.isArray(entry)) {
      throw new Error(`Server "${name}" is not an object.`);
    }
    const spec = entry as Record<string, unknown>;
    const parsedServer: AddMCPServerInput = { name };

    if (typeof spec.command === "string") parsedServer.command = spec.command;
    if (Array.isArray(spec.args)) {
      parsedServer.args = spec.args.filter(
        (arg): arg is string => typeof arg === "string",
      );
    }
    if (typeof spec.url === "string") parsedServer.url = spec.url;
    parsedServer.env = stringMap(spec.env);
    parsedServer.headers = stringMap(spec.headers);

    if (!parsedServer.command && !parsedServer.url) {
      throw new Error(`Server "${name}" has neither a command nor a url.`);
    }
    return parsedServer;
  });
}

export interface AddMCPServerInput {
  name: string;
  command?: string;
  args?: string[];
  url?: string;
  env?: Record<string, string>;
  headers?: Record<string, string>;
}

function stringMap(value: unknown): Record<string, string> | undefined {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  const entries = Object.entries(value as Record<string, unknown>).filter(
    (entry): entry is [string, string] => typeof entry[1] === "string",
  );
  if (entries.length === 0) return undefined;
  return Object.fromEntries(entries);
}
