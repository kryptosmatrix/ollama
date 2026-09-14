import { afterEach, describe, expect, it, vi } from "vitest";
import { respondToToolApproval } from "./api";

function mockFetch(status: number) {
  const fetchMock = vi.fn(async () =>
    Promise.resolve({ ok: status >= 200 && status < 300, status } as Response),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("respondToToolApproval", () => {
  it("posts the answer to the chat's approval endpoint", async () => {
    const fetchMock = mockFetch(200);

    await respondToToolApproval("chat-1", "chat-1:abc", {
      allow: true,
      remember: true,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/v1/chat/chat-1/approval");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toEqual({
      approvalId: "chat-1:abc",
      allow: true,
      remember: true,
      rememberAll: false,
    });
  });

  it("never sends a refusal as any kind of agreement", async () => {
    const fetchMock = mockFetch(200);

    await respondToToolApproval("chat-1", "chat-1:abc", {
      allow: false,
      remember: true,
      rememberAll: true,
    });

    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(init.body as string)).toEqual({
      approvalId: "chat-1:abc",
      allow: false,
      remember: false,
      rememberAll: false,
    });
  });

  it("reports a call that is no longer waiting rather than doing nothing", async () => {
    mockFetch(409);
    await expect(
      respondToToolApproval("chat-1", "chat-1:gone", { allow: true }),
    ).rejects.toThrow("no longer waiting");
  });

  it("reports any other failure", async () => {
    mockFetch(500);
    await expect(
      respondToToolApproval("chat-1", "chat-1:abc", { allow: true }),
    ).rejects.toThrow("500");
  });
});
