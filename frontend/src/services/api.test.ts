import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, api } from "./api";
afterEach(() => vi.unstubAllGlobals());
describe("api client", () => {
  it("sends credentials and decodes successful JSON", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    vi.stubGlobal("fetch", fetchMock);
    await expect(api<{ ok: boolean }>("/api/v1/example")).resolves.toEqual({
      ok: true,
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/example",
      expect.objectContaining({ credentials: "include" }),
    );
  });
  it("turns the standard error envelope into ApiError", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({
              error: { code: "FORBIDDEN", message: "Access denied" },
            }),
            { status: 403 },
          ),
        ),
    );
    const request = api("/api/v1/restricted");
    await expect(request).rejects.toBeInstanceOf(ApiError);
    await expect(request).rejects.toMatchObject({
      code: "FORBIDDEN",
      status: 403,
    });
  });
  it("supports empty successful responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    await expect(
      api("/api/v1/logout", { method: "POST" }),
    ).resolves.toBeUndefined();
  });
});
