import { figure, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it('muestra cada medio de pago, lo recibido y el cambio', () => {
    const receipt = renderReceipt();

    expect(figure(receipt, 'Tarjeta')).toBe('$ 30.000');
    expect(figure(receipt, 'Efectivo')).toBe('$ 30.000');
    expect(figure(receipt, 'Recibido')).toBe('$ 60.000');
    expect(figure(receipt, 'Cambio')).toBe('$ 3.655');
  });
});
