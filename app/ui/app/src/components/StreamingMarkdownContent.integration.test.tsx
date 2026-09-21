import { beforeAll, describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import StreamingMarkdownContent from "./StreamingMarkdownContent";
import { highlighterPromise } from "@/lib/highlighter";

beforeAll(async () => {
  await highlighterPromise;
});

function render(content: string, pageStack?: string[]) {
  return renderToStaticMarkup(
    <StreamingMarkdownContent
      content={content}
      browserToolResult={pageStack ? { page_stack: pageStack } : undefined}
    />,
  );
}

describe("StreamingMarkdownContent with real Streamdown and Shiki", () => {
  it("renders distinct upstream Markdown rather than fixed or empty output", () => {
    const first = render("Alpha **important**.");
    const second = render("Bravo *different*.");
    expect(first).toContain("Alpha");
    expect(first).toContain("<strong>important</strong>");
    expect(first).not.toContain("Bravo");
    expect(second).toContain("Bravo");
    expect(second).toContain("<em>different</em>");
    expect(second).not.toContain("Alpha");
  });

  it("renders the input code through the loaded syntax highlighter", () => {
    const first = render("```javascript\nconst answer = 42;\n```");
    const second = render("```javascript\nreturn false;\n```");
    expect(first).toContain("answer");
    expect(first).toContain("42");
    expect(first).toContain("color:");
    expect(first).not.toContain("false");
    expect(second).toContain("return");
    expect(second).toContain("false");
    expect(second).not.toContain("answer");
  });

  it("uses the citation cursor to select the correct upstream page", () => {
    const html = render("Evidence 【1†L2-L4】.", [
      "https://first.example/wrong-page",
      "https://second.example/right-page",
    ]);
    expect(html).toContain('href="https://second.example/right-page"');
    expect(html).not.toContain('href="https://first.example/wrong-page"');
    expect(html).toContain("[1]");
  });

  it("does not invent a link for a missing citation page", () => {
    const html = render("Evidence 【9†source】.");
    expect(html).toContain("[9]");
    expect(html).not.toContain("href=");
  });

  it("does not turn model-provided images or raw HTML into embedded resources", () => {
    const html = render(
      '![diagram](https://image.example/tracker)\n\n<iframe src="https://frame.example/embed"></iframe>',
    );
    expect(html).not.toContain("<img");
    expect(html).not.toContain("<iframe");
    expect(html).not.toContain('src="https://image.example');
    expect(html).not.toContain('src="https://frame.example');
    expect(html).toContain("diagram");
  });
});
