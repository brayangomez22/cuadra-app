import type { RouteObject } from 'react-router';
import { routes } from './router';
import { paths } from './routes/paths';

// Every pattern of paths, flattened: a feature whose routes the router forgets
// leaves its URLs without a screen.
function patterns(tree: object): string[] {
  return Object.values(tree).flatMap((value: unknown) => {
    if (typeof value === 'function' && 'pattern' in value) return [String(value.pattern)];
    return typeof value === 'object' && value !== null ? patterns(value) : [];
  });
}

function routePaths(list: RouteObject[]): string[] {
  return list.flatMap((route) => [...(route.path ? [route.path] : []), ...routePaths(route.children ?? [])]);
}

describe('router', () => {
  it('cada URL de paths tiene su ruta en el router', () => {
    expect(routePaths(routes).sort()).toEqual(patterns(paths).sort());
  });
});
