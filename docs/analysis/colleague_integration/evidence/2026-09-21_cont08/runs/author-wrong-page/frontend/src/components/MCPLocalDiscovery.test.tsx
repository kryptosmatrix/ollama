import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { DiscoveredRow } from "./MCPLocalDiscovery";
import { MCPDiscoveredServer } from "@/gotypes";

function server(
  overrides: Partial<MCPDiscoveredServer> = {},
): MCPDiscoveredServer {
  return new MCPDiscoveredServer({
    name: "files",
    sources: ["Claude Desktop"],
    paths: [
      "/Users/x/Library/Application Support/Claude/claude_desktop_config.json",
    ],
    runs: "uvx mcp-server-files",
    origin: "config",
    command: "uvx",
    args: ["mcp-server-files"],
    ...overrides,
  });
}

describe("DiscoveredRow", () => {
  it("shows what would be added, verbatim, and where it came from", () => {
    const markup = renderToStaticMarkup(
      <DiscoveredRow server={server()} onAdd={() => {}} />,
    );
    expect(markup).toContain("uvx mcp-server-files");
    expect(markup).toContain("Claude Desktop");
    expect(markup).toContain("Add…");
  });

  // What discovery left behind has to be visible before the click, not after.
  it("shows a credential that was not copied", () => {
    const markup = renderToStaticMarkup(
      <DiscoveredRow
        server={server({
          notes: ["WEATHER_API_KEY held a value that looks like a credential"],
        })}
        onAdd={() => {}}
      />,
    );
    expect(markup).toContain("WEATHER_API_KEY");
  });

  // An entry that cannot be added is listed with its reason rather than
  // dropped: one that vanishes reads as a discovery that missed it.
  it("offers no button for something that cannot be added", () => {
    const markup = renderToStaticMarkup(
      <DiscoveredRow
        server={server({ problem: "command is required", command: "" })}
        onAdd={() => {}}
      />,
    );
    expect(markup).toContain("command is required");
    expect(markup).not.toContain("Add…");
  });
});

describe("one server registered in several applications", () => {
  // Which applications is what tells a user they are looking at their own
  // setup rather than a duplicate the page invented.
  it("names every application configured with it", () => {
    const markup = renderToStaticMarkup(
      <DiscoveredRow
        server={server({
          name: "agora-memory",
          sources: ["Claude Code", "Cursor"],
          paths: ["/Users/x/.claude.json", "/Users/x/.cursor/mcp.json"],
        })}
        onAdd={() => {}}
      />,
    );
    expect(markup).toContain("Claude Code, Cursor");
    expect(markup).toContain("/Users/x/.claude.json");
    expect(markup).toContain("/Users/x/.cursor/mcp.json");
  });
});
