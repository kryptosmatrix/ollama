import { describe, expect, it } from "vitest";
import { nextPanel } from "./mcpPanels";

describe("nextPanel", () => {
  it("opens the one that was clicked", () => {
    expect(nextPanel(null, "registry")).toBe("registry");
  });

  // The bug this exists to prevent: with two panels open at once, the second
  // one renders below the first, off the bottom of a window that does not
  // scroll, and its button reads as doing nothing.
  it("replaces whatever was open rather than stacking", () => {
    expect(nextPanel("registry", "paste")).toBe("paste");
    expect(nextPanel("paste", "local")).toBe("local");
  });

  it("closes the panel when its own button is clicked again", () => {
    expect(nextPanel("registry", "registry")).toBeNull();
  });
});
