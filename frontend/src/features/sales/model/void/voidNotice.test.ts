import { plain } from '@/test/fixtures';
import { aSale } from '../../testing/fixtures';
import { cashToReturn, voidNotices } from './voidNotice';

const fabricSale = aSale({
  lines: [{ ...aSale().lines[0]!, product_name: 'Camiseta talla M, azul' }],
});

describe('cashToReturn', () => {
  it.each([
    { name: 'ferretería: el efectivo menos el cambio', sale: aSale(), want: '11900.00' },
    {
      name: 'ropa: solo la parte en efectivo de un pago mixto',
      sale: aSale({
        payments: [
          { method: 'cash', amount: '5000.00' },
          { method: 'card', amount: '6900.00' },
        ],
        paid: '11900.00',
        change: '0.00',
      }),
      want: '5000.00',
    },
    {
      name: 'sin efectivo no hay nada que devolver',
      sale: aSale({ payments: [{ method: 'nequi', amount: '11900.00' }], paid: '11900.00', change: '0.00' }),
      want: '0.00',
    },
  ])('$name', ({ sale, want }) => {
    expect(cashToReturn(sale)).toBe(want);
  });
});

describe('voidNotices', () => {
  const texts = (...args: Parameters<typeof voidNotices>) => voidNotices(...args).map(plain);

  it('siempre avisa que el stock vuelve al inventario', () => {
    expect(texts(fabricSale, 'open')[0]).toBe('Lo vendido vuelve al inventario.');
  });

  it('con la caja abierta pide devolver el efectivo y no registrar nada', () => {
    expect(texts(aSale(), 'open')).toContain('Devuelva $ 11.900 en efectivo. La caja deja de contar esta venta.');
  });

  it('con la caja cerrada pide registrar el egreso en la caja actual', () => {
    expect(texts(aSale(), 'closed')).toContain(
      'Devuelva $ 11.900 en efectivo y regístrelo como egreso de la caja actual: la caja de esta venta ya se cerró.',
    );
  });

  it('una venta sin caja también pide registrar el egreso', () => {
    expect(texts(aSale(), 'none')).toContain(
      'Devuelva $ 11.900 en efectivo y regístrelo como egreso de la caja actual: esta venta no quedó en ninguna caja.',
    );
  });

  it('un medio distinto del efectivo se devuelve por fuera de Cuadra', () => {
    const sale = aSale({ payments: [{ method: 'card', amount: '11900.00' }], paid: '11900.00', change: '0.00' });
    expect(texts(sale, 'none')).toEqual([
      'Lo vendido vuelve al inventario.',
      'Devuelva $ 11.900 de Tarjeta por ese mismo medio: Cuadra no mueve ese dinero.',
    ]);
  });

  it('una venta a crédito cancela la deuda y no pide devolver ese valor', () => {
    const sale = aSale({
      customer_id: '01920000-0000-7000-8000-000000000301',
      payments: [{ method: 'credit', amount: '11900.00' }],
      paid: '11900.00',
      change: '0.00',
    });
    expect(texts(sale, 'none')).toEqual([
      'Lo vendido vuelve al inventario.',
      'Se cancela la deuda de $ 11.900 que dejó esta venta en la cuenta del cliente.',
    ]);
  });
});
