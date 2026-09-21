import { readFileSync } from "node:fs";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";
import {
  Chat,
  ChatEvent,
  ChatInfo,
  ChatResponse,
  ChatsResponse,
  File,
  Message,
  MCPServersResponse,
  Settings,
  SettingsResponse,
  ToolCall,
  ToolFunction,
} from "@/gotypes";
import { getChat } from "@/api";
import StreamingMarkdownContent from "@/components/StreamingMarkdownContent";
import { highlighterPromise } from "@/lib/highlighter";

const firstTime = "2026-01-02T03:04:05.000Z";
const secondTime = "2026-01-03T04:05:06.000Z";

// Synthetic Go-JSON-shaped responses; only the external HTTP transport is simulated.
function responseFor(id: string, content: string) {
  return {
    chat: {
      id,
      title: `Conversation ${id}`,
      created_at: firstTime,
      messages: [
        {
          role: "assistant",
          content,
          thinking: "",
          stream: false,
          created_at: firstTime,
          updated_at: secondTime,
          thinkingTimeStart: firstTime,
          thinkingTimeEnd: secondTime,
          attachments: [{ filename: "evidence.txt", data: "cHJlc2VydmU=" }],
          tool_calls: [
            {
              type: "function",
              function: {
                name: "lookup",
                arguments: '{"query":"retained"}',
                result: { count: 0, found: false, values: [null, "retained"] },
              },
            },
          ],
        },
        {
          role: "user",
          content: `Follow-up ${id}`,
          thinking: "",
          stream: false,
          created_at: secondTime,
          updated_at: secondTime,
        },
      ],
    },
  };
}

beforeAll(async () => {
  await highlighterPromise;
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("generated model production consumers", () => {
  it("materialises nested chat models through the real API consumer", async () => {
    const wire = responseFor("alpha", "Alpha **important**.");
    const fetch = vi.fn(async () => new Response(JSON.stringify(wire)));
    vi.stubGlobal("fetch", fetch);

    const response = await getChat("alpha");
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining("/api/v1/chat/alpha"));
    expect(response).toBeInstanceOf(ChatResponse);
    expect(response.chat).toBeInstanceOf(Chat);
    expect(response.chat.id).toBe("alpha");
    expect(response.chat.messages).toHaveLength(2);
    const first = response.chat.messages[0];
    expect(first).toBeInstanceOf(Message);
    expect(first.attachments?.[0]).toBeInstanceOf(File);
    expect(first.attachments?.[0].filename).toBe("evidence.txt");
    expect(first.tool_calls?.[0]).toBeInstanceOf(ToolCall);
    expect(first.tool_calls?.[0].function).toBeInstanceOf(ToolFunction);
    expect(first.tool_calls?.[0].function.result).toEqual({
      count: 0,
      found: false,
      values: [null, "retained"],
    });
    expect(response.chat.messages[1].content).toBe("Follow-up alpha");
  });

  it("delivers distinct decoded content to the real Markdown renderer", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) =>
        new Response(
          JSON.stringify(
            url.endsWith("/alpha")
              ? responseFor("alpha", "Alpha **important**.")
              : responseFor("beta", "Beta *different*."),
          ),
        ),
      ),
    );
    const first = await getChat("alpha");
    const second = await getChat("beta");
    const render = (response: ChatResponse) =>
      renderToStaticMarkup(
        <StreamingMarkdownContent content={response.chat.messages[0].content} />,
      );
    const alpha = render(first);
    const beta = render(second);
    expect(alpha).toContain("Alpha");
    expect(alpha).toMatch(/<span\b[^>]*data-streamdown="strong"[^>]*>important<\/span>/);
    expect(alpha).not.toContain("Beta");
    expect(beta).toContain("Beta");
    expect(beta).toMatch(/<em\b[^>]*>different<\/em>/);
    expect(beta).not.toContain("Alpha");
  });

  it("keeps string and object constructor inputs equivalent without mutating their values", () => {
    const wire = responseFor("copy", "Preserve café.");
    const before = JSON.stringify(wire);
    const object = new ChatResponse(wire);
    const string = new ChatResponse(before);
    expect(string).toEqual(object);
    expect(string.chat.messages[0]).toBeInstanceOf(Message);
    expect(JSON.stringify(wire)).toBe(before);
  });

  it("preserves optional nested values without inventing arrays or objects", () => {
    expect(new Message({ content: "bare" }).tool_calls).toBeUndefined();
    expect(new Message({ tool_calls: null }).tool_calls).toBeNull();
    expect(new Message({ tool_calls: [] }).tool_calls).toEqual([]);
    expect(new MCPServersResponse({ servers: [] }).servers).toEqual([]);
  });

  it("performs declared Date transforms on real generated classes", () => {
    const info = new ChatInfo({ createdAt: firstTime, updatedAt: secondTime });
    expect(info.createdAt).toBeInstanceOf(Date);
    expect(info.createdAt.toISOString()).toBe(firstTime);
    expect(info.updatedAt.toISOString()).toBe(secondTime);
    const event = new ChatEvent({
      eventName: "thinking",
      thinkingTimeStart: firstTime,
      thinkingTimeEnd: secondTime,
    });
    expect(event.thinkingTimeStart?.toISOString()).toBe(firstTime);
    expect(event.thinkingTimeEnd?.toISOString()).toBe(secondTime);
    expect(new ChatEvent({ eventName: "done" }).thinkingTimeStart).toBeUndefined();
  });

  it("preserves primitive and structured JSON payloads", () => {
    for (const result of [null, false, 0, "", "text", [0, false], { nested: [null, 7] }]) {
      const tool = new ToolFunction({ name: "lookup", arguments: "{}", result });
      expect(tool.result).toEqual(result);
      const event = new ChatEvent({
        eventName: "tool_result",
        toolResultData: result,
        approvalArgs: { retained: result },
      });
      expect(event.toolResultData).toEqual(result);
      expect(event.approvalArgs?.retained).toEqual(result);
    }
  });

  it("retains nested Settings construction and false or zero values", () => {
    const response = new SettingsResponse({
      settings: { ContextLength: 0, Tools: false, WorkingDir: "" },
    });
    expect(response.settings).toBeInstanceOf(Settings);
    expect(response.settings.ContextLength).toBe(0);
    expect(response.settings.Tools).toBe(false);
    expect(response.settings.WorkingDir).toBe("");
  });

  it("retains map conversion and its existing reference behaviour", () => {
    const source = { first: { id: "map", createdAt: firstTime, updatedAt: secondTime } };
    const converted = new ChatsResponse().convertValues(source, ChatInfo, true);
    expect(converted).toBe(source);
    expect(source.first).toBeInstanceOf(ChatInfo);
    expect(source.first.createdAt).toEqual(new Date(firstTime));
  });

  it("does not swallow malformed JSON constructor errors", () => {
    expect(() => new ChatResponse("{")).toThrow(SyntaxError);
    expect(() => new Message("not JSON")).toThrow(SyntaxError);
  });

  it("ships generated declarations without explicit-any syntax", () => {
    const source = readFileSync(new URL("../../codegen/gotypes.gen.ts", import.meta.url), "utf8");
    const file = ts.createSourceFile("gotypes.gen.ts", source, ts.ScriptTarget.Latest, true);
    const positions: number[] = [];
    const visit = (node: ts.Node) => {
      if (node.kind === ts.SyntaxKind.AnyKeyword) positions.push(node.getStart(file));
      ts.forEachChild(node, visit);
    };
    visit(file);
    expect(positions).toEqual([]);
  });
});
