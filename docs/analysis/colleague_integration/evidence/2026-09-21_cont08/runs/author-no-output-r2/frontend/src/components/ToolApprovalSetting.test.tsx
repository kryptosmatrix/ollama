import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AUTO_APPROVE_LABEL, ToolApprovalSetting } from "./ToolApprovalSetting";

function render(checked: boolean) {
  return renderToStaticMarkup(
    <ToolApprovalSetting checked={checked} onChange={() => {}} />,
  );
}

describe("ToolApprovalSetting", () => {
  it("names the switch for what it does", () => {
    expect(render(false)).toContain(AUTO_APPROVE_LABEL);
  });

  it("shows the stored setting in the switch, not a fixed state", () => {
    expect(render(true)).toContain('aria-checked="true"');
    expect(render(false)).toContain('aria-checked="false"');
  });

  it("says what changes while it is on, because the user is choosing to stop being asked", () => {
    expect(render(true)).toContain("without asking");
    expect(render(false)).toContain("wait for you in the chat");
  });
});
