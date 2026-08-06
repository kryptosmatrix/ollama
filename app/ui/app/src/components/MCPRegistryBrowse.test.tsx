import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import {
  RegistryInstallConfirmation,
  RegistryResult,
} from "./MCPRegistryBrowse";
import { MCPRegistryEntry } from "@/gotypes";

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
    variables: ["${env:WEATHER_API_KEY}"],
    ...overrides,
  });
}

describe("RegistryResult", () => {
  const render = (overrides: Partial<MCPRegistryEntry> = {}) =>
    renderToStaticMarkup(
      <RegistryResult entry={entry(overrides)} onInstall={() => {}} />,
    );

  it("shows the command line, not just a name and a description", () => {
    expect(render()).toContain("npx -y @example/weather-mcp@1.2.0");
  });

  it("always names the publisher", () => {
    expect(render()).toContain("io.github.example");
  });

  it("shows the repository when the publisher gave one", () => {
    expect(render()).toContain("https://github.com/example/weather");
  });

  it("offers no install button for an entry Ollama cannot run", () => {
    const markup = render({
      installable: false,
      runs: "",
      reason: "Ollama does not know how to run a cargo package",
    });
    expect(markup).not.toContain("Add…");
    expect(markup).toContain("does not know how to run");
  });

  it("renders registry-supplied text as text, never as markup", () => {
    const markup = render({
      title: "evil<script>alert(1)</script>",
      description: "<img src=x onerror=alert(1)>",
    });
    expect(markup).not.toContain("<script>");
    expect(markup).not.toContain("<img src=x");
    expect(markup).toContain("&lt;script&gt;");
  });
});

describe("RegistryInstallConfirmation", () => {
  const render = (overrides: Partial<MCPRegistryEntry> = {}) =>
    renderToStaticMarkup(
      <RegistryInstallConfirmation
        entry={entry(overrides)}
        onConfirm={() => {}}
        onCancel={() => {}}
      />,
    );

  it("shows the exact command line before anything is written", () => {
    expect(render()).toContain("npx -y @example/weather-mcp@1.2.0");
  });

  it("names the values the user must set", () => {
    expect(render()).toContain("WEATHER_API_KEY");
  });

  it("says the server will not run until it is approved", () => {
    const markup = render();
    expect(markup).toContain("not approved");
    expect(markup).toContain("will not run it until you approve");
  });

  it("names the server it would be added as", () => {
    expect(render()).toContain("weather");
  });
});
