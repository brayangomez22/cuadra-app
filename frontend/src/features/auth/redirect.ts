import type { Location } from 'react-router';

// The page a visitor asked for before being sent to the login, kept in the
// location state by RequireAuth.
export type FromState = { from?: Pick<Location, 'pathname' | 'search'> };

export function redirectTarget(state: unknown): string {
  if (typeof state === 'object' && state !== null && 'from' in state) {
    const { from } = state as FromState;
    if (from && typeof from.pathname === 'string' && from.pathname.startsWith('/')) {
      return from.pathname + (from.search ?? '');
    }
  }
  return '/';
}
