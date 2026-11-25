import { useEffect, useState, type ReactNode } from 'react';
import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getErrorStatus, getRetryAfterMs } from '@/shared/api';

/** The other tabs of this browser listen here and reload what a change in this one made stale. */
const CHANGES_CHANNEL = 'dona:data-changed';

declare module '@tanstack/react-query' {
  interface Register {
    // `announcedByServer`: the other tabs hear of this change from the server's events.
    mutationMeta: { announcedByServer?: boolean };
  }
}

function shouldRetry(failureCount: number, error: unknown) {
  const status = getErrorStatus(error);
  // "Too many requests" passes by itself. Without a retry a refused refetch would leave old data on
  // screen with nothing to say so.
  if (status === 429) return failureCount < 3;
  // Any other 4xx answer will not change on retry.
  if (status && status >= 400 && status < 500) return false;
  return failureCount < 2;
}

function retryDelay(failureCount: number, error: unknown) {
  return getRetryAfterMs(error) ?? Math.min(1000 * 2 ** failureCount, 30_000);
}

function createClient(onChanged: () => void) {
  return new QueryClient({
    mutationCache: new MutationCache({
      onSuccess: (_data, _variables, _context, mutation) => {
        // Telling the other tabs about such a change too would reload them twice, and nginx counts.
        if (!mutation.meta?.announcedByServer) onChanged();
      },
    }),
    defaultOptions: {
      queries: {
        retry: shouldRetry,
        retryDelay,
        staleTime: 30_000,
        // Back on the tab, the seller sees what is true now: it may have changed on another device.
        refetchOnWindowFocus: 'always',
      },
    },
  });
}

export function QueryProvider({ children }: { children: ReactNode }) {
  const [{ client, channel }] = useState(() => {
    const channel = typeof BroadcastChannel === 'undefined' ? null : new BroadcastChannel(CHANGES_CHANNEL);
    return { client: createClient(() => channel?.postMessage('changed')), channel };
  });

  useEffect(() => {
    if (!channel) return;
    // A hidden tab only marks its data stale: it reloads when the seller comes back to it.
    const reload = () =>
      void client.invalidateQueries({
        refetchType: document.visibilityState === 'visible' ? 'active' : 'none',
      });
    channel.addEventListener('message', reload);
    return () => channel.removeEventListener('message', reload);
  }, [client, channel]);

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
