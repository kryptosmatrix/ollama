import { describe, expect, it } from "vitest";
import { MCPDiscoveredServer } from "@/gotypes";
import { addRequestForDiscovered } from "./mcpDiscovery";

function server(
  overrides: Partial<MCPDiscoveredServer> = {},
): MCPDiscoveredServer {
  return new MCPDiscoveredServer({
    name: "files",
    sources: ["Claude Desktop"],
    runs: "uvx mcp-server-files",
    origin: "config",
    command: "uvx",
    args: ["mcp-server-files"],
    ...overrides,
  });
}

describe("addRequestForDiscovered", () => {
  it("carries the specification through rather than rebuilding it", () => {
    expect(addRequestForDiscovered(server())).toEqual({
      name: "files",
      command: "uvx",
      args: ["mcp-server-files"],
      env: undefined,
    });
  });

  it("carries an environment reference across untouched", () => {
    const request = addRequestForDiscovered(
      server({ env: { WEATHER_API_KEY: "${env:WEATHER_API_KEY}" } }),
    );
    expect(request.env).toEqual({ WEATHER_API_KEY: "${env:WEATHER_API_KEY}" });
  });

  it("adds a server that answered on this machine by its address", () => {
    const request = addRequestForDiscovered(
      server({
        command: "",
        args: undefined,
        url: "http://127.0.0.1:3000/mcp",
        runs: "http://127.0.0.1:3000/mcp",
        origin: "listening",
      }),
    );
    expect(request).toEqual({
      name: "files",
      url: "http://127.0.0.1:3000/mcp",
      headers: undefined,
    });
  });

  // The problem is the reason it must not be added; adding it anyway would
  // write a configuration Ollama has already said it cannot use.
  it("refuses an entry that carries a problem", () => {
    expect(() =>
      addRequestForDiscovered(server({ problem: "command is required" })),
    ).toThrow("command is required");
  });
});
