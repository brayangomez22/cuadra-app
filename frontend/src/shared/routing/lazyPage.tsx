import type { ComponentType } from 'react';
import type { RouteObject } from 'react-router';
import { Spinner } from '@/shared/ui';

/**
 * A route whose screen loads with its first visit, in the chunk of its
 * feature: a cashier who only sells never downloads the catalog screens, and
 * a signed-in user never downloads the login forms.
 */
export function lazyPage(load: () => Promise<ComponentType>): Pick<RouteObject, 'lazy' | 'HydrateFallback'> {
  return {
    lazy: async () => ({ Component: await load() }),
    // Shown while the section loads on the first page of a visit.
    HydrateFallback: () => <Spinner label="Cargando…" />,
  };
}
