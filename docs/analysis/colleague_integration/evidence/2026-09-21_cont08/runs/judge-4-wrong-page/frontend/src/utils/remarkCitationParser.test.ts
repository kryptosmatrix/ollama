import { describe, expect, it } from "vitest";
import { remark } from "remark";
import remarkCitationParser from "./remarkCitationParser";

function parse(content: string) {
  const processor = remark().use(remarkCitationParser);
  return processor.runSync(processor.parse(content));
}

describe("remarkCitationParser production pipeline", () => {
  it("carries the exact range and surrounding text into the citation node", () => {
    expect(parse("Before 【1†L25-L30】 after.").children).toMatchObject([
      {
        type: "paragraph",
        children: [
          { type: "text", value: "Before " },
          {
            type: "custom-citation",
            data: {
              hName: "ol-citation",
              hProperties: { cursor: "1", start: "25", end: "30" },
            },
          },
          { type: "text", value: " after." },
        ],
      },
    ]);
  });

  it("preserves the distinct generic citation payload", () => {
    expect(parse("【12†source】").children).toMatchObject([
      {
        type: "paragraph",
        children: [
          {
            type: "custom-citation",
            data: { hName: "ol-citation", hProperties: { cursor: "12" } },
          },
        ],
      },
    ]);
  });

  it("coalesces adjacent equal cursors without dropping a different cursor", () => {
    const paragraph = parse("【1†L1-L2】【1†L3-L4】【2†source】").children[0];
    expect(paragraph).toMatchObject({
      type: "paragraph",
      children: [
        { type: "custom-citation", data: { hProperties: { cursor: "1" } } },
        { type: "custom-citation", data: { hProperties: { cursor: "2" } } },
      ],
    });
    expect(paragraph && "children" in paragraph && paragraph.children).toHaveLength(2);
  });

  it("does not coalesce citations separated by ordinary text", () => {
    expect(parse("【3†source】 and 【3†source】").children).toMatchObject([
      {
        type: "paragraph",
        children: [
          { type: "custom-citation", data: { hProperties: { cursor: "3" } } },
          { type: "text", value: " and " },
          { type: "custom-citation", data: { hProperties: { cursor: "3" } } },
        ],
      },
    ]);
  });

  it("leaves code and malformed delimiters alone", () => {
    expect(parse("`【1†L2-L3】` 【not-a-cursor†source】").children).toMatchObject([
      {
        type: "paragraph",
        children: [
          { type: "inlineCode", value: "【1†L2-L3】" },
          { type: "text", value: " 【not-a-cursor†source】" },
        ],
      },
    ]);
  });

  it("preserves an empty document without manufacturing a citation", () => {
    expect(parse("").children).toEqual([]);
  });
});
