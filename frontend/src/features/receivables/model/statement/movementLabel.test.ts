import { movementLabel } from './movementLabel';

describe('movementLabel', () => {
  it.each([
    [{ kind: 'charge' } as const, 'Venta a crédito'],
    [{ kind: 'payment', method: 'cash' } as const, 'Abono · Efectivo'],
    [{ kind: 'payment', method: 'nequi' } as const, 'Abono · Nequi'],
    [{ kind: 'payment' } as const, 'Abono'],
    [{ kind: 'cancellation' } as const, 'Venta a crédito anulada'],
  ])('nombra %o', (movement, label) => {
    expect(movementLabel(movement)).toBe(label);
  });
});
