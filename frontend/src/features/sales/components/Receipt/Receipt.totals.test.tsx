import { within } from '@testing-library/react';
import { plain } from '@/test/fixtures';
import { aSale } from '../../testing/fixtures';
import { figure, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it('muestra el IVA agrupado por tasa', () => {
    const receipt = renderReceipt();

    const table = within(receipt.getByRole('table', { name: 'IVA por tasa' }));
    const rows = table
      .getAllByRole('row')
      .slice(1)
      .map((row) => plain(row.textContent));
    expect(rows).toEqual(['0 %$ 5.000$ 0', '5 %$ 20.000$ 1.000', '19 %$ 25.500$ 4.845']);
  });

  it('los totales coinciden con la venta', () => {
    const receipt = renderReceipt();

    expect(figure(receipt, 'Subtotal')).toBe('$ 52.500');
    expect(figure(receipt, 'Descuentos')).toBe('-$ 2.000');
    expect(figure(receipt, 'IVA')).toBe('$ 5.845');
    expect(figure(receipt, 'Total')).toBe('$ 56.345');
  });

  it('sin descuentos no muestra la fila de descuentos', () => {
    const receipt = renderReceipt(aSale());

    expect(receipt.queryByText('Descuentos', { selector: 'dt' })).not.toBeInTheDocument();
    expect(figure(receipt, 'Total')).toBe('$ 11.900');
  });
});
