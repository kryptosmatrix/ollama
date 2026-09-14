import { afterEach, describe, expect, it, vi } from "vitest";
import { speakMessageChunk } from "./api";

describe("speakMessageChunk", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not treat a JSON body as audio", async () => {
    const fetchMock = vi.fn(async () =>
      Promise.resolve({
        ok: true,
        status: 200,
        headers: {
          get: (name: string) =>
            name.toLowerCase() === "content-type" ? "application/json" : null,
        },
        blob: async () => new Blob(["{}"], { type: "application/json" }),
      } as unknown as Response),
    );
    vi.stubGlobal("fetch", fetchMock);
    await expect(speakMessageChunk("hello", 0)).rejects.toThrow("not audio");
  });
});
