import { useQueryClient } from '@tanstack/react-query';
import { clearAuthToken, readAuthToken, writeAuthToken } from '@/shared/lib';

/** Whether a seller is signed in. The token's expiry is checked by the API (401 signs out). */
export function hasSession() {
  return readAuthToken() !== null;
}

/**
 * Starts and ends a session. Both drop every cached query, so one seller never sees
 * another shop's data that was loaded before.
 */
export function useSession() {
  const queryClient = useQueryClient();

  return {
    start(token: string) {
      queryClient.clear();
      writeAuthToken(token);
    },
    end() {
      clearAuthToken();
      queryClient.clear();
    },
  };
}
