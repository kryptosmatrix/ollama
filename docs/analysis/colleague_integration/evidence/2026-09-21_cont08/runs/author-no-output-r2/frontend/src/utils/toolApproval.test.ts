import { describe, expect, it } from "vitest";
import type { ChatEvent } from "@/gotypes";
import {
  approvalArguments,
  approvalRequestBody,
  MAX_ARGUMENT_LENGTH,
  pendingApprovalFromEvent,
} from "./toolApproval";

type ApprovalEvent = Pick<
  ChatEvent,
  "eventName" | "toolName" | "approvalId" | "approvalScope" | "approvalArgs"
>;

function event(overrides: Partial<ApprovalEvent> = {}): ApprovalEvent {
  return {
    eventName: "tool_approval",
    toolName: "files__read",
    approvalId: "chat-1:abc",
    approvalScope: "files__read",
    approvalArgs: { path: "/etc/passwd" },
    ...overrides,
  } as ApprovalEvent;
}

describe("pendingApprovalFromEvent", () => {
  it("reads an approval out of its event", () => {
    expect(pendingApprovalFromEvent(event())).toEqual({
      approvalId: "chat-1:abc",
      toolName: "files__read",
      scope: "files__read",
      args: { path: "/etc/passwd" },
    });
  });

  it("ignores every other kind of event", () => {
    for (const name of ["chat", "tool", "tool_result", "done"]) {
      expect(
        pendingApprovalFromEvent(event({ eventName: name as never })),
      ).toBeNull();
    }
  });

  it("ignores an approval with no identifier, which could never be answered", () => {
    expect(
      pendingApprovalFromEvent(event({ approvalId: undefined })),
    ).toBeNull();
    expect(pendingApprovalFromEvent(event({ approvalId: "" }))).toBeNull();
  });

  it("falls back to the tool name when no scope is given", () => {
    const approval = pendingApprovalFromEvent(event({ approvalScope: "" }));
    expect(approval?.scope).toBe("files__read");
  });

  it("tolerates an approval with no arguments", () => {
    const approval = pendingApprovalFromEvent(
      event({ approvalArgs: undefined }),
    );
    expect(approval?.args).toEqual({});
  });
});

describe("approvalArguments", () => {
  it("shows arguments in a stable order", () => {
    const rendered = approvalArguments({ zulu: 1, alpha: 2, mike: 3 });
    expect(rendered.map((a) => a.name)).toEqual(["alpha", "mike", "zulu"]);
  });

  it("renders strings as themselves and everything else as JSON", () => {
    const rendered = approvalArguments({
      path: "/etc/passwd",
      count: 3,
      flags: ["a", "b"],
      nested: { deep: true },
    });
    const byName = Object.fromEntries(rendered.map((a) => [a.name, a.value]));
    expect(byName.path).toBe("/etc/passwd");
    expect(byName.count).toBe("3");
    expect(byName.flags).toBe('["a","b"]');
    expect(byName.nested).toBe('{"deep":true}');
  });

  it("strips control characters, which a value could use to forge the layout", () => {
    const [rendered] = approvalArguments({
      path: "safe \u001b[31mnot safe\u0000\r",
    });
    expect(rendered.value).not.toContain("\u001b");
    expect(rendered.value).not.toContain("\u0000");
    expect(rendered.value).not.toContain("\r");
    expect(rendered.value).toContain("safe");
  });

  it("caps an enormous value so it cannot push the buttons off the screen", () => {
    const [rendered] = approvalArguments({ blob: "x".repeat(50_000) });
    expect(rendered.value.length).toBeLessThan(MAX_ARGUMENT_LENGTH + 40);
    expect(rendered.value).toContain("truncated");
  });

  it("survives a value that cannot be serialised", () => {
    const cyclic: Record<string, unknown> = {};
    cyclic.self = cyclic;
    expect(() => approvalArguments({ cyclic })).not.toThrow();
  });
});

describe("approvalRequestBody", () => {
  it("sends a plain allowance", () => {
    expect(approvalRequestBody("chat-1:abc", { allow: true })).toEqual({
      approvalId: "chat-1:abc",
      allow: true,
      remember: false,
      rememberAll: false,
    });
  });

  it("sends remember when the user asked to be remembered", () => {
    expect(
      approvalRequestBody("chat-1:abc", { allow: true, remember: true }),
    ).toEqual({
      approvalId: "chat-1:abc",
      allow: true,
      remember: true,
      rememberAll: false,
    });
  });

  it("treats remember-all as remembering too", () => {
    expect(
      approvalRequestBody("chat-1:abc", { allow: true, rememberAll: true }),
    ).toEqual({
      approvalId: "chat-1:abc",
      allow: true,
      remember: true,
      rememberAll: true,
    });
  });

  it("never describes a refusal as any kind of agreement", () => {
    expect(
      approvalRequestBody("chat-1:abc", {
        allow: false,
        remember: true,
        rememberAll: true,
      }),
    ).toEqual({
      approvalId: "chat-1:abc",
      allow: false,
      remember: false,
      rememberAll: false,
    });
  });
});
