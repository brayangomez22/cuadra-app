import { plain } from '@/test/fixtures';
import { figure, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it('muestra el nombre del negocio, el NIT, la sede, el cajero y el consecutivo', () => {
    const receipt = renderReceipt();

    expect(receipt.getByRole('heading', { name: 'Ferretería El Tornillo' })).toBeInTheDocument();
    expect(receipt.getByText('NIT 900123456-8')).toBeInTheDocument();
    expect(receipt.getByText('Bodega principal')).toBeInTheDocument();
    expect(figure(receipt, 'Venta N.º')).toBe('42');
    expect(figure(receipt, 'Cajero')).toBe('Ana Gómez');
    // Bogotá time: 15:00 UTC is 10:00 a. m.
    expect(plain(figure(receipt, 'Fecha'))).toMatch(/^11\/10\/2026, 10:00 a\. ?m\.$/);
  });
});
