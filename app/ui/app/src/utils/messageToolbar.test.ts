import { describe, expect, it } from "vitest";
import { showAssistantMessageToolbar } from "./messageToolbar";

const assistant = {
  role: "assistant",
  content: "Hello",
};

describe("showAssistantMessageToolbar", () => {
  it("hides the speaker while the message is still streaming", () => {
    expect(showAssistantMessageToolbar(assistant, true)).toBe(false);
    expect(showAssistantMessageToolbar(assistant, false)).toBe(true);
  });

  it("hides on tool chrome and empty content", () => {
    expect(
      showAssistantMessageToolbar({ ...assistant, tool_call: { name: "x" } }, false),
    ).toBe(false);
    expect(
      showAssistantMessageToolbar(
        { ...assistant, tool_calls: [{ name: "x" }] },
        false,
      ),
    ).toBe(false);
    expect(showAssistantMessageToolbar({ role: "assistant", content: "  " }, false)).toBe(
      false,
    );
    expect(showAssistantMessageToolbar({ role: "user", content: "Hello" }, false)).toBe(
      false,
    );
  });
});
