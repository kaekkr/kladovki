/**
 * The Android app (Capacitor) runs the SPA at https://localhost and calls the
 * API on another site. Cross-site, the HttpOnly session cookie is not sent
 * back (SameSite + WebView third-party cookie blocking): login "worked" but
 * every data request was 401. In that case the app asks the server for the
 * token and sends it as "Authorization: Bearer". The website keeps the cookie.
 */
export function usesTokenAuth(apiUrl: string, pageOrigin: string): boolean {
  return new URL(apiUrl).origin !== pageOrigin;
}

/** Only our own API ever receives the token. */
export function attachesToken(requestUrl: string, apiUrl: string): boolean {
  return requestUrl === apiUrl || requestUrl.startsWith(apiUrl.replace(/\/$/, '') + '/');
}

const KEY = 'kladovki.session';

/** Token storage (WebView localStorage persists across app restarts). */
export const tokenStore = {
  get(): string | null {
    try {
      return localStorage.getItem(KEY);
    } catch {
      return null;
    }
  },
  set(token: string): void {
    try {
      localStorage.setItem(KEY, token);
    } catch {
      /* storage unavailable: the session lasts until the app closes */
    }
  },
  clear(): void {
    try {
      localStorage.removeItem(KEY);
    } catch {
      /* ignore */
    }
  },
};
