import { describe, expect, it } from "vitest";
import { voicesAfterKeyChange } from "./SpeechSettings";

describe("voicesAfterKeyChange", () => {
  const previous = [{ voice_id: "v1", name: "Ada" }];

  it("drops the previous voice list when the key is gone", () => {
    expect(voicesAfterKeyChange(false, previous)).toEqual([]);
  });

  it("keeps the list only while a key is present", () => {
    expect(voicesAfterKeyChange(true, previous)).toEqual(previous);
  });
});
