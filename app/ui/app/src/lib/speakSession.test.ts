import { describe, expect, it } from "vitest";
import {
  isPlayableSpeechMIME,
  pauseDoesNotRewind,
  pausePlayback,
  type AudioHandle,
} from "./speakSession";

function fakeAudio(at: number): AudioHandle {
  return {
    currentTime: at,
    src: "",
    onended: null,
    onerror: null,
    play: async () => {},
    pause() {},
  };
}

describe("speak pause", () => {
  it("does not reset currentTime", () => {
    const audio = fakeAudio(4.2);
    pausePlayback(audio);
    expect(audio.currentTime).toBe(4.2);
    expect(pauseDoesNotRewind(audio)).toBe(true);
  });

  it("fails a pause that rewinds — stop labelled as pause", () => {
    const rewinding: AudioHandle = {
      currentTime: 4.2,
      src: "",
      onended: null,
      onerror: null,
      play: async () => {},
      pause() {
        this.currentTime = 0;
      },
    };
    expect(pauseDoesNotRewind(rewinding)).toBe(false);
  });
});

describe("playable MIME", () => {
  it("accepts mpeg and rejects json", () => {
    expect(isPlayableSpeechMIME("audio/mpeg")).toBe(true);
    expect(isPlayableSpeechMIME("audio/mpeg; charset=binary")).toBe(true);
    expect(isPlayableSpeechMIME("application/json")).toBe(false);
    expect(isPlayableSpeechMIME("text/plain")).toBe(false);
  });
});
