import { cleanUrl } from './cleanUrl';

describe('cleanUrl', () => {
  it.each([
    ['https://app.cuadra.co/clientes?q=ana+gomez', 'https://app.cuadra.co/clientes'],
    ['https://app.cuadra.co/productos?q=bota#lista', 'https://app.cuadra.co/productos'],
    ['https://app.cuadra.co/vender', 'https://app.cuadra.co/vender'],
    ['/ventas#arriba', '/ventas'],
  ])('%s → %s', (url, clean) => {
    expect(cleanUrl(url)).toBe(clean);
  });
});
