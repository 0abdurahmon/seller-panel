import { useEffect } from "react";
import { EventStreamError, followOrderEvents } from "@/shared/api";

const MIN_DELAY_MS = 1_000;
const MAX_DELAY_MS = 30_000;
// A stream that lived this long was healthy, so the next reconnect starts from the shortest delay again.
// Opening alone proves nothing: a stream that is closed at once must keep backing off.
const HEALTHY_AFTER_MS = 10_000;
// The first change is reported at once; more within this window share one report at its end. A refetch
// is about 10 requests and nginx allows 10 a second from one address, so the window cannot be much shorter.
const REFRESH_WINDOW_MS = 2_000;

function sleep(ms: number, signal: AbortSignal) {
  return new Promise<void>((resolve) => {
    const done = () => {
      clearTimeout(timer);
      signal.removeEventListener("abort", done);
      resolve();
    };
    const timer = setTimeout(done, ms);
    signal.addEventListener("abort", done);
  });
}

/**
 * Follows the shop's order events and calls `onChange` when an order is created or changes status.
 * Events are not replayed, so a reconnect calls it too. Mount it once for the whole signed-in app.
 *
 * Only a visible page holds a stream. Over HTTP/1.1 a browser has six connections per site:
 * streams kept by background tabs, or by pages kept for Back, would use them all and every
 * other request would hang.
 */
export function useOrderEvents(onChange: () => void) {
  useEffect(() => {
    let windowTimer: ReturnType<typeof setTimeout> | undefined;
    let changedAgain = false;
    const refresh = () => {
      if (windowTimer !== undefined) {
        changedAgain = true;
        return;
      }
      onChange();
      windowTimer = setTimeout(() => {
        windowTimer = undefined;
        if (!changedAgain) return;
        changedAgain = false;
        refresh();
      }, REFRESH_WINDOW_MS);
    };

    // True after any time without a stream (hidden, a failed first attempt, a drop).
    let mayHaveMissed = document.visibilityState !== "visible";
    let current: AbortController | undefined;

    async function follow(signal: AbortSignal) {
      let delay = MIN_DELAY_MS;
      while (!signal.aborted) {
        let openedAt: number | undefined;
        let retryAfterMs: number | undefined;
        try {
          await followOrderEvents(signal, {
            onOpen: () => {
              openedAt = Date.now();
              if (mayHaveMissed) refresh();
              mayHaveMissed = false;
            },
            onEvent: refresh,
          });
        } catch (error) {
          if (error instanceof EventStreamError) {
            // The client has already sent the seller to the login page.
            if (error.status === 401) return;
            // Too many streams for the shop: wait as long as the server asks.
            if (error.retryAfter) retryAfterMs = error.retryAfter * 1000;
          }
        }
        if (signal.aborted) return;
        mayHaveMissed = true;
        if (openedAt !== undefined && Date.now() - openedAt >= HEALTHY_AFTER_MS) delay = MIN_DELAY_MS;
        await sleep(retryAfterMs ?? delay, signal);
        delay = Math.min(delay * 2, MAX_DELAY_MS);
      }
    }

    const stop = () => {
      if (!current) return;
      current.abort();
      current = undefined;
      mayHaveMissed = true;
    };
    const sync = () => {
      if (document.visibilityState !== "visible") return stop();
      if (current) return;
      current = new AbortController();
      void follow(current.signal);
    };

    document.addEventListener("visibilitychange", sync);
    window.addEventListener("pagehide", stop);
    window.addEventListener("pageshow", sync);
    sync();

    return () => {
      document.removeEventListener("visibilitychange", sync);
      window.removeEventListener("pagehide", stop);
      window.removeEventListener("pageshow", sync);
      stop();
      clearTimeout(windowTimer);
    };
  }, [onChange]);
}
