import { describe, expect, it } from "vitest";
import { MCPRegistryEntry } from "@/gotypes";
import {
  installRequest,
  installableEntries,
  presentRegistryEntry,
} from "./mcpRegistry";

function entry(overrides: Partial<MCPRegistryEntry> = {}): MCPRegistryEntry {
  return new MCPRegistryEntry({
    name: "io.github.example/weather",
    title: "Weather",
    description: "Forecasts",
    publisher: "io.github.example",
    repository: "https://github.com/example/weather",
    installable: true,
    transport: "stdio",
    runs: "npx -y @example/weather-mcp@1.2.0",
    suggestedName: "weather",
    command: "npx",
    args: ["-y", "@example/weather-mcp@1.2.0"],
    env: { WEATHER_API_KEY: "${env:WEATHER_API_KEY}" },
    variables: ["${env:WEATHER_API_KEY}"],
    ...overrides,
  });
}

describe("presentRegistryEntry", () => {
  it("always shows the publisher, which is the only provenance on offer", () => {
    expect(presentRegistryEntry(entry()).publisher).toBe("io.github.example");
  });

  it("falls back to the identifier when there is no title", () => {
    expect(presentRegistryEntry(entry({ title: "" })).heading).toBe(
      "io.github.example/weather",
    );
  });

  it("carries the exact command line through for display", () => {
    expect(presentRegistryEntry(entry()).runs).toBe(
      "npx -y @example/weather-mcp@1.2.0",
    );
  });

  it("carries the reason an entry cannot be installed", () => {
    const shown = presentRegistryEntry(
      entry({
        installable: false,
        runs: "",
        reason: 'Ollama does not know how to run a "cargo" package',
      }),
    );
    expect(shown.installable).toBe(false);
    expect(shown.reason).toContain("cargo");
    expect(shown.runs).toBe("");
  });
});

describe("installRequest", () => {
  it("installs the resolved fields, not a re-parse of the displayed string", () => {
    expect(installRequest(entry())).toEqual({
      name: "weather",
      command: "npx",
      args: ["-y", "@example/weather-mcp@1.2.0"],
      env: { WEATHER_API_KEY: "${env:WEATHER_API_KEY}" },
    });
  });

  it("installs a hosted entry by its address", () => {
    const request = installRequest(
      entry({
        transport: "http",
        command: "",
        args: undefined,
        env: undefined,
        runs: "https://mcp.example.com/v1",
        url: "https://mcp.example.com/v1",
        headers: { Authorization: "${env:AUTHORIZATION}" },
      }),
    );
    expect(request).toEqual({
      name: "weather",
      url: "https://mcp.example.com/v1",
      headers: { Authorization: "${env:AUTHORIZATION}" },
    });
  });

  it("honours a name the user chose over the suggestion", () => {
    expect(installRequest(entry(), "my-weather").name).toBe("my-weather");
  });

  it("refuses an entry Ollama cannot run, with the registry's reason", () => {
    expect(() =>
      installRequest(
        entry({ installable: false, reason: "unknown ecosystem" }),
      ),
    ).toThrow("unknown ecosystem");
  });

  it("refuses an entry with no name", () => {
    expect(() => installRequest(entry({ suggestedName: "" }))).toThrow(
      "needs a name",
    );
    expect(() => installRequest(entry(), "   ")).toThrow("needs a name");
  });

  it("refuses a local entry with no command, rather than inventing one", () => {
    expect(() => installRequest(entry({ command: "" }))).toThrow(
      "no command to install",
    );
  });

  it("refuses a hosted entry with no address", () => {
    expect(() => installRequest(entry({ transport: "http", url: "" }))).toThrow(
      "no address to install",
    );
  });

  it("never carries a literal secret, only environment references", () => {
    const request = installRequest(entry());
    for (const value of Object.values(request.env ?? {})) {
      expect(value).toMatch(/^\$\{env:[A-Z0-9_]+\}$/);
    }
  });
});

describe("installableEntries", () => {
  const listing = (name: string, installable: boolean) =>
    new MCPRegistryEntry({ name, publisher: "io.example", installable });

  it("offers only what Ollama can install", () => {
    const { shown } = installableEntries([
      listing("io.example/a", true),
      listing("io.example/b", false),
    ]);
    expect(shown.map((e) => e.name)).toEqual(["io.example/a"]);
  });

  // Silently dropping results would misrepresent the registry, so the count
  // survives for the page to say out loud.
  it("counts what it left out", () => {
    expect(
      installableEntries([
        listing("io.example/a", true),
        listing("io.example/b", false),
        listing("io.example/c", false),
      ]).hidden,
    ).toBe(2);
  });

  it("hides nothing when everything is installable", () => {
    expect(installableEntries([listing("io.example/a", true)]).hidden).toBe(0);
  });
});
