import { useState, type ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getErrorStatus } from '@/shared/api';

function shouldRetry(failureCount: number, error: unknown) {
  // A 4xx answer will not change on retry (and 429 has its own Retry-After).
  const status = getErrorStatus(error);
  if (status && status >= 400 && status < 500) return false;
  return failureCount < 2;
}

export function QueryProvider({ children }: { children: ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: shouldRetry, staleTime: 30_000, refetchOnWindowFocus: false },
        },
      }),
  );
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
