// Session holds the access token in memory only: never in localStorage, so an XSS
// cannot read it. After a reload the first request gets a 401 and authFetch renews
// the session with the refresh cookie.
export type Session = {
  getAccessToken(): string | null;
  setAccessToken(token: string): void;
  // end forgets the token and notifies the listeners (e.g. to go to the login).
  end(): void;
  // onEnd registers a listener and returns the function that removes it.
  onEnd(listener: () => void): () => void;
};

export function createSession(): Session {
  let accessToken: string | null = null;
  const listeners = new Set<() => void>();

  return {
    getAccessToken: () => accessToken,
    setAccessToken: (token) => {
      accessToken = token;
    },
    end: () => {
      accessToken = null;
      listeners.forEach((listener) => listener());
    },
    onEnd: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}
