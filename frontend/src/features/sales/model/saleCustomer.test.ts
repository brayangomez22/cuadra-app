import { loadSaleCustomer, saveSaleCustomer } from './saleCustomer';

const key = 'cuadra.pos.cart.user-1';

beforeEach(() => sessionStorage.clear());

describe('saleCustomer', () => {
  it.each([
    // Ferretería: un maestro de obra; almacén de ropa: una empresa mayorista.
    { id: 'c-1', name: 'Carlos Pérez', document: 'CC 1020304050' },
    { id: 'c-2', name: 'Confecciones Sol SAS', document: 'NIT 901234567-7' },
  ])('guarda y recupera el cliente $name de la venta', (customer) => {
    saveSaleCustomer(key, customer);
    expect(loadSaleCustomer(key)).toEqual(customer);
  });

  it('sin cliente la venta es a consumidor final', () => {
    expect(loadSaleCustomer(key)).toBeNull();
  });

  it('quitar el cliente lo borra del almacenamiento', () => {
    saveSaleCustomer(key, { id: 'c-1', name: 'Carlos Pérez', document: 'CC 1020304050' });
    saveSaleCustomer(key, null);
    expect(loadSaleCustomer(key)).toBeNull();
  });

  it.each(['no es json', JSON.stringify({ id: 'c-1' }), JSON.stringify(['c-1'])])(
    'descarta un cliente dañado (%s)',
    (stored) => {
      sessionStorage.setItem(`${key}.customer`, stored);
      expect(loadSaleCustomer(key)).toBeNull();
    },
  );
});
