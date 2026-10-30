import { matchRoutes } from 'react-router';
import { routes } from '@/app/router';

/**
 * Loads ahead the lazy screens of the given URLs, for a beforeAll: the first
 * import of a big page (the POS, the product form) takes seconds in a cold
 * worker and would eat the first test's timeout.
 */
export function preloadPages(...urls: string[]) {
  return async () => {
    const matches = urls.flatMap((url) => matchRoutes(routes, url) ?? []);
    const loads = matches.flatMap(({ route }) => (typeof route.lazy === 'function' ? [route.lazy()] : []));
    await Promise.all(loads);
  };
}
