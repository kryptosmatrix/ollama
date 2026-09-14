import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { ToolApproval } from "./ToolApproval";
import type { PendingApproval } from "@/utils/toolApproval";

function approval(overrides: Partial<PendingApproval> = {}): PendingApproval {
  return {
    approvalId: "chat-1:abc",
    toolName: "files__read",
    scope: "files__read",
    args: { path: "/etc/passwd", lines: 40 },
    ...overrides,
  };
}

function render(props: Partial<Parameters<typeof ToolApproval>[0]> = {}) {
  return renderToStaticMarkup(
    <ToolApproval
      approval={approval()}
      onAnswer={() => {}}
      {...(props as object)}
    />,
  );
}

describe("ToolApproval", () => {
  it("names the tool being asked about", () => {
    expect(render()).toContain("files__read");
  });

  it("shows the arguments, because the user is agreeing to this call", () => {
    const markup = render();
    expect(markup).toContain("path");
    expect(markup).toContain("/etc/passwd");
    expect(markup).toContain("lines");
    expect(markup).toContain("40");
  });

  it("offers allow once, always allow, and decline", () => {
    const markup = render();
    expect(markup).toContain("Allow once");
    expect(markup).toContain("Always allow");
    expect(markup).toContain("Decline");
  });

  it("scopes the always-allow option to the tool it is about", () => {
    const markup = renderToStaticMarkup(
      <ToolApproval
        approval={approval({ scope: "github__create_issue" })}
        onAnswer={() => {}}
      />,
    );
    expect(markup).toContain("Always allow github__create_issue");
  });

  it("renders an error from a failed answer", () => {
    const markup = render({
      error: "That tool call is no longer waiting for an answer.",
    });
    expect(markup).toContain("no longer waiting");
  });

  it("disables the buttons while an answer is in flight, so it cannot be sent twice", () => {
    const markup = render({ isAnswering: true });
    expect(markup).toContain("disabled");
  });

  it("renders server-supplied text as text, never as markup", () => {
    const markup = renderToStaticMarkup(
      <ToolApproval
        approval={approval({
          toolName: "evil<script>alert(1)</script>",
          args: { path: "<img src=x onerror=alert(1)>" },
        })}
        onAnswer={() => {}}
      />,
    );
    expect(markup).not.toContain("<script>");
    expect(markup).not.toContain("<img src=x");
    expect(markup).toContain("&lt;script&gt;");
  });

  it("renders without arguments", () => {
    const markup = renderToStaticMarkup(
      <ToolApproval approval={approval({ args: {} })} onAnswer={() => {}} />,
    );
    expect(markup).toContain("files__read");
  });
});
