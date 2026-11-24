const STORAGE_KEY = 'access_token';

export function readAuthToken(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

export function writeAuthToken(token: string) {
  try {
    localStorage.setItem(STORAGE_KEY, token);
  } catch {
    // Storage can be unavailable (private mode); the seller logs in again next time.
  }
}

export function clearAuthToken() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clear.
  }
}
