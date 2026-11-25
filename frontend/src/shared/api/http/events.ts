import { clearAuthToken, readAuthToken } from "@/shared/lib";
import { API_URL } from "./index";

// The server pings every 25 s. Two missed pings and the connection is dead even if nobody closed it.
const SILENCE_LIMIT_MS = 60_000;

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
async function readEvents(
  body: ReadableStream<Uint8Array>,
  onData: () => void,
  onEvent: (name: string) => void,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return;
    onData();
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
 * Follows `GET /orders/events` until the stream ends (the token expired, the server stopped, the
 * connection dropped or went silent) or `signal` aborts. Resolves when the stream ends, rejects when
 * it cannot be opened or breaks. Events are not replayed, so the caller reloads after a reconnect.
 *
 * `fetch` and not `EventSource`: the stream needs the Authorization header.
 */
export async function followOrderEvents(
  signal: AbortSignal,
  handlers: { onOpen: () => void; onEvent: (name: string) => void },
) {
  const connection = new AbortController();
  const close = () => connection.abort();
  signal.addEventListener("abort", close);
  if (signal.aborted) close();

  let silence: ReturnType<typeof setTimeout> | undefined;
  const expectData = () => {
    clearTimeout(silence);
    silence = setTimeout(close, SILENCE_LIMIT_MS);
  };

  try {
    expectData();
    const token = readAuthToken();
    const response = await fetch(`${API_URL}/orders/events`, {
      headers: { Accept: "text/event-stream", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      signal: connection.signal,
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
    await readEvents(response.body, expectData, handlers.onEvent);
  } finally {
    clearTimeout(silence);
    signal.removeEventListener("abort", close);
  }
}
