import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { PauseIcon, SpeakerWaveIcon } from "@heroicons/react/24/outline";

function glyph(playing: boolean) {
  return renderToStaticMarkup(
    playing ? (
      <PauseIcon className="h-7 w-7" />
    ) : (
      <SpeakerWaveIcon className="h-7 w-7" />
    ),
  );
}

describe("speak glyphs", () => {
  it("uses outline pause while playing and speaker otherwise", () => {
    const playing = glyph(true);
    const idle = glyph(false);
    expect(playing).not.toEqual(idle);
    expect(playing).toContain("h-7 w-7");
    expect(idle).toContain("h-7 w-7");
  });
});
