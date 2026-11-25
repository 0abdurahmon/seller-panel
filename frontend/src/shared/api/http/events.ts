import { clearAuthToken, readAuthToken } from "@/shared/lib";
import { API_URL } from "./index";

/** The stream could not be opened or was refused; `retryAfter` is in seconds when the server sent it. */
export class EventStreamError extends Error {
  status: number;
  retryAfter?: number;

  constructor(status: number, retryAfter?: number) {
    super(`event stream failed: ${status}`);
    this.status = status;
    this.retryAfter = retryAfter;
  }
}

/** Reads the `event:` name of every message from a text/event-stream body; comments (`: ping`) are skipped. */
async function readEvents(body: ReadableStream<Uint8Array>, onEvent: (name: string) => void) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return;
    buffer += decoder.decode(value, { stream: true });
    // A message ends with an empty line; keep the unfinished tail for the next chunk.
    const messages = buffer.split(/\r?\n\r?\n/);
    buffer = messages.pop() ?? "";
    for (const message of messages) {
      const name = message.split(/\r?\n/).find((line) => line.startsWith("event:"));
      if (name) onEvent(name.slice("event:".length).trim());
    }
  }
}

/**
 * Follows `GET /orders/events` until the stream ends (the token expired, the server stopped or
 * the connection dropped) or `signal` aborts. Resolves when the stream ends, rejects when it
 * cannot be opened. Events are not replayed, so the caller reloads its data after a reconnect.
 *
 * `fetch` and not `EventSource`: the stream needs the Authorization header.
 */
export async function followOrderEvents(
  signal: AbortSignal,
  handlers: { onOpen: () => void; onEvent: (name: string) => void },
) {
  const token = readAuthToken();
  const response = await fetch(`${API_URL}/orders/events`, {
    headers: { Accept: "text/event-stream", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    signal,
  });

  if (response.status === 401) {
    // The same as the rest of the client: an expired or foreign token sends the seller to the login page.
    clearAuthToken();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      window.location.href = "/login";
    }
  }
  if (!response.ok || !response.body) {
    const retryAfter = Number(response.headers.get("Retry-After"));
    throw new EventStreamError(response.status, retryAfter > 0 ? retryAfter : undefined);
  }

  handlers.onOpen();
  await readEvents(response.body, handlers.onEvent);
}
