import { redirectTarget } from './redirect';

describe('redirectTarget', () => {
  it.each([
    {
      name: 'vuelve a la ruta pedida con su query',
      state: { from: { pathname: '/productos', search: '?q=tornillo' } },
      want: '/productos?q=tornillo',
    },
    { name: 'vuelve a la ruta pedida sin query', state: { from: { pathname: '/ventas/pos' } }, want: '/ventas/pos' },
    { name: 'sin estado lleva al inicio', state: undefined, want: '/' },
    { name: 'con un estado que no es un objeto lleva al inicio', state: 'productos', want: '/' },
    { name: 'ignora una ruta que no empieza por /', state: { from: { pathname: 'https://otro.sitio' } }, want: '/' },
    { name: 'ignora un from sin pathname', state: { from: { search: '?talla=M' } }, want: '/' },
    { name: 'ignora un from nulo', state: { from: null }, want: '/' },
  ])('$name', ({ state, want }) => {
    expect(redirectTarget(state)).toBe(want);
  });
});
