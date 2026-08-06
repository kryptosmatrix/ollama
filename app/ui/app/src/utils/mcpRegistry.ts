import type { MCPRegistryEntry } from "@/gotypes";
import type { AddMCPServerInput } from "./mcpServers";

/**
 * What to show for one registry result.
 *
 * The registry is an open-publish metadata service, so provenance is the
 * publisher's namespace and their repository link and nothing more. Both are
 * surfaced rather than summarised away: they are all a person has to go on.
 */
export interface RegistryPresentation {
  /** The display name, falling back to the identifier when there is no title. */
  heading: string;
  /** The publisher namespace, always shown. */
  publisher: string;
  /** Repository URL, when the publisher gave one. */
  repository: string;
  /** True when Ollama can build a command line for this entry. */
  installable: boolean;
  /** Why it cannot be installed, when it cannot. */
  reason: string;
  /** The exact command line or URL an install would write. */
  runs: string;
  /** Values the user must set in their environment before it will work. */
  variables: string[];
}

export function presentRegistryEntry(
  entry: MCPRegistryEntry,
): RegistryPresentation {
  return {
    heading: entry.title?.trim() || entry.name,
    publisher: entry.publisher,
    repository: entry.repository ?? "",
    installable: entry.installable,
    reason: entry.reason ?? "",
    runs: entry.runs ?? "",
    variables: entry.variables ?? [],
  };
}

/**
 * Turns a resolved registry entry into the request that adds it.
 *
 * The command line is deliberately not reconstructed here. The server resolved
 * it and the user read that exact string; rebuilding it in the browser would
 * be a second implementation that could drift from the one they agreed to, so
 * the resolved fields are carried straight through.
 */
export function installRequest(
  entry: MCPRegistryEntry,
  name?: string,
): AddMCPServerInput {
  if (!entry.installable) {
    throw new Error(entry.reason || "Ollama cannot install this entry.");
  }

  const serverName = (name ?? entry.suggestedName ?? "").trim();
  if (serverName === "") {
    throw new Error("This entry needs a name before it can be added.");
  }

  // The resolved specification is carried structurally, so nothing here has
  // to take apart the string the user read. Runs is for reading; these are
  // what gets stored, and they are the same resolution.
  if (entry.transport === "http") {
    if (!entry.url) {
      throw new Error("This entry has no address to install.");
    }
    return {
      name: serverName,
      url: entry.url,
      headers: entry.headers ?? undefined,
    };
  }

  if (!entry.command) {
    throw new Error("This entry has no command to install.");
  }
  return {
    name: serverName,
    command: entry.command,
    args: entry.args ?? undefined,
    env: entry.env ?? undefined,
  };
}
