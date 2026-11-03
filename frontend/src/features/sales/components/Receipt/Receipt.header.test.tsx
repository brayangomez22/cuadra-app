import { plain } from '@/test/fixtures';
import { figure, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it('muestra el nombre del negocio, el NIT, la sede, el cajero y el consecutivo', () => {
    const receipt = renderReceipt();

    expect(receipt.getByRole('heading', { name: 'Ferretería El Tornillo' })).toBeInTheDocument();
    expect(receipt.getByText('NIT 900123456-8')).toBeInTheDocument();
    expect(receipt.getByText('Calle 45 # 12-30, Medellín')).toBeInTheDocument();
    expect(receipt.getByText('Tel. 604 444 1234')).toBeInTheDocument();
    expect(receipt.getByText('Bodega principal')).toBeInTheDocument();
    expect(figure(receipt, 'Venta N.º')).toBe('42');
    expect(figure(receipt, 'Cajero')).toBe('Ana Gómez');
    // Bogotá time: 15:00 UTC is 10:00 a. m.
    expect(plain(figure(receipt, 'Fecha'))).toMatch(/^11\/10\/2026, 10:00 a\. ?m\.$/);
  });

  it('sin dirección ni teléfono registrados no deja renglones vacíos', () => {
    // Almacén de ropa que todavía no registra sus datos de contacto.
    const store = { name: 'Calzado La Horma', nit: '900123456-8', address: '', phone: '' };
    const receipt = renderReceipt(undefined, undefined, store);

    expect(receipt.queryByText(/^Tel\./)).not.toBeInTheDocument();
    expect(receipt.getByRole('heading', { name: 'Calzado La Horma' }).parentElement?.children).toHaveLength(3);
  });
});
