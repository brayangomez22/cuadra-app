import { paths } from './paths';

describe('paths', () => {
  it('arma la URL de editar un producto con su ID', () => {
    expect(paths.products.edit('01920000-0000-7000-8000-000000000101')).toBe(
      '/productos/01920000-0000-7000-8000-000000000101/editar',
    );
  });

  it('arma el kardex con la variante y la sede en la query', () => {
    expect(paths.inventory.kardex('p1', { variante: 'v1', sede: 's1' })).toBe(
      '/productos/p1/kardex?variante=v1&sede=s1',
    );
  });

  it('omite de la query los valores vacíos', () => {
    expect(paths.inventory.receipt({ producto: 'p1', variante: 'v1', sede: undefined })).toBe(
      '/inventario/entrada?producto=p1&variante=v1',
    );
    expect(paths.inventory.receipt()).toBe('/inventario/entrada');
  });

  it('codifica los parámetros de la URL', () => {
    expect(paths.sales.cashReport('a/b c')).toBe('/caja/a%2Fb%20c');
    expect(paths.inventory.kardex('p1', { sede: 'a&b' })).toBe('/productos/p1/kardex?sede=a%26b');
  });

  it('arma las ventas de un día y el detalle de una venta', () => {
    expect(paths.sales.list({ fecha: '2026-10-11', sede: 's1' })).toBe('/ventas?fecha=2026-10-11&sede=s1');
    expect(paths.sales.detail('v1')).toBe('/ventas/v1');
  });

  it('expone el patrón que usa el router', () => {
    expect(paths.products.edit.pattern).toBe('/productos/:productId/editar');
    expect(paths.settings.units.pattern).toBe('/configuracion/unidades');
    expect(paths.home()).toBe('/');
  });
});
