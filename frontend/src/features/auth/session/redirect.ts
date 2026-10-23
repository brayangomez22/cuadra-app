import type { Location } from 'react-router';
import { paths } from '@/app/routes/paths';

// The page a visitor asked for before being sent to the login, kept in the
// location state by RequireAuth.
export type FromState = { from?: Pick<Location, 'pathname' | 'search'> };

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

// The state comes from the history entry, so it is checked instead of trusted;
// only a path of this app is followed.
export function redirectTarget(state: unknown): string {
  const from = isObject(state) ? state.from : undefined;
  if (!isObject(from) || typeof from.pathname !== 'string' || !from.pathname.startsWith('/')) return paths.home();
  return from.pathname + (typeof from.search === 'string' ? from.search : '');
}
