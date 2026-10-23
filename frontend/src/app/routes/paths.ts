// The app's URLs, in one place. The router builds its routes from each
// pattern and the code links with the builders, so a renamed URL breaks tsc
// instead of a link. Tests keep the literal URL on purpose: they check the
// address the user sees.

// Params turns '/caja/:sessionId' into [sessionId: string], in pattern order.
type Params<P extends string> = P extends `${string}:${string}/${infer Rest}`
  ? [string, ...Params<`/${Rest}`>]
  : P extends `${string}:${string}`
    ? [string]
    : [];

type Query<K extends readonly string[]> = Partial<Record<K[number], string | undefined>>;

/** A URL builder: call it with the pattern's params (and its query) to get the URL. */
export type Route<P extends string, K extends readonly string[]> = ((
  ...args: K extends readonly [] ? Params<P> : [...Params<P>, query?: Query<K>]
) => string) & { pattern: P };

// route builds the URL of a pattern. Params are encoded, and query values that
// are empty or undefined are left out, in the order the route lists them.
function route<P extends string, const K extends readonly string[] = []>(pattern: P, queryKeys?: K): Route<P, K> {
  const build = (...args: unknown[]) => {
    let index = 0;
    const path = pattern.replace(/:[^/]+/g, () => encodeURIComponent(String(args[index++])));
    const query = (args[index] ?? {}) as Record<string, string | undefined>;
    const search = new URLSearchParams();
    for (const key of queryKeys ?? []) {
      const value = query[key];
      if (value) search.set(key, value);
    }
    const qs = search.toString().replaceAll('+', '%20');
    return qs ? `${path}?${qs}` : path;
  };
  return Object.assign(build, { pattern });
}

/** Every URL of the app, in Spanish: the user sees them. */
export const paths = {
  home: route('/'),
  login: route('/ingresar'),
  signUp: route('/registro'),
  products: {
    list: route('/productos'),
    new: route('/productos/nuevo'),
    edit: route('/productos/:productId/editar'),
  },
  inventory: {
    stock: route('/inventario'),
    receipt: route('/inventario/entrada', ['producto', 'variante', 'sede']),
    kardex: route('/productos/:productId/kardex', ['variante', 'sede']),
  },
  sales: {
    pos: route('/vender'),
    cash: route('/caja'),
    closeCash: route('/caja/cerrar'),
    cashReport: route('/caja/:sessionId'),
  },
  settings: {
    home: route('/configuracion'),
    general: route('/configuracion/general'),
    units: route('/configuracion/unidades'),
    attributes: route('/configuracion/atributos'),
    variantAxes: route('/configuracion/ejes'),
    modules: route('/configuracion/modulos'),
  },
  dev: {
    ui: route('/dev/ui'),
  },
};
