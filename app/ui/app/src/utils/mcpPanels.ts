/**
 * Which of the three ways to add a server is open.
 *
 * One at a time, not three independent toggles. With independent toggles,
 * opening "Add from configuration" while a registry search is on screen put
 * the field below hundreds of pixels of results, off the bottom of a window
 * that does not scroll — so the button read as doing nothing at all. Only one
 * panel open means the one you asked for is where you are looking.
 */
export type MCPPanel = "registry" | "paste" | "local" | null;

/** Clicking the button for the open panel closes it; anything else replaces it. */
export function nextPanel(
  current: MCPPanel,
  clicked: Exclude<MCPPanel, null>,
): MCPPanel {
  return current === clicked ? null : clicked;
}
