import type { MCPDiscoveredServer } from "@/gotypes";
import type { AddMCPServerInput } from "./mcpServers";

/**
 * Turns a server found on this machine into the request that adds it.
 *
 * The specification is carried straight through rather than rebuilt from the
 * command line the user read. Rebuilding it here would be a second resolution
 * of the same thing, and the one the user agreed to must be the one that gets
 * written.
 */
export function addRequestForDiscovered(
  server: MCPDiscoveredServer,
): AddMCPServerInput {
  if (server.problem) {
    throw new Error(server.problem);
  }

  if (server.url) {
    return {
      name: server.name,
      url: server.url,
      headers: server.headers ?? undefined,
    };
  }

  if (!server.command) {
    throw new Error("This server has no command to add.");
  }
  return {
    name: server.name,
    command: server.command,
    args: server.args ?? undefined,
    env: server.env ?? undefined,
  };
}
