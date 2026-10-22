import { render, screen, within } from '@testing-library/react';
import { aSale, plain, type Sale } from '@/test/fixtures';
import { Receipt } from './Receipt';

const business = { name: 'Ferretería El Tornillo', nit: '900123456-8' };

// A sale with IVA at three rates; two lines at 19 % add up in one group.
function aMixedSale(overrides: Partial<Sale> = {}): Sale {
  return aSale({
    number: 42,
    lines: [
      {
        variant_id: '01920000-0000-7000-8000-000000000211',
        product_id: '01920000-0000-7000-8000-000000000111',
        product_name: 'Martillo de uña 16 oz',
        quantity: '1',
        unit_factor: '1',
        base_quantity: '1',
        unit_price: '20000.00',
        price_rule: 'variant_price',
        tax_rate: '0.19',
        discount: '2000.00',
        gross: '20000.00',
        base: '18000.00',
        tax: '3420.00',
        total: '21420.00',
      },
      {
        variant_id: '01920000-0000-7000-8000-000000000212',
        product_id: '01920000-0000-7000-8000-000000000112',
        product_name: 'Cable eléctrico 12 AWG',
        quantity: '2.5',
        unit_factor: '1',
        base_quantity: '2.5',
        unit_price: '3000.00',
        price_rule: 'variant_price',
        tax_rate: '0.19',
        discount: '0.00',
        gross: '7500.00',
        base: '7500.00',
        tax: '1425.00',
        total: '8925.00',
      },
      {
        variant_id: '01920000-0000-7000-8000-000000000213',
        product_id: '01920000-0000-7000-8000-000000000113',
        product_name: 'Abono triple 15 1kg',
        quantity: '2',
        unit_factor: '1',
        base_quantity: '2',
        unit_price: '10000.00',
        price_rule: 'variant_price',
        tax_rate: '0.05',
        discount: '0.00',
        gross: '20000.00',
        base: '20000.00',
        tax: '1000.00',
        total: '21000.00',
      },
      {
        variant_id: '01920000-0000-7000-8000-000000000214',
        product_id: '01920000-0000-7000-8000-000000000114',
        product_name: 'Arena lavada bulto',
        quantity: '1',
        unit_factor: '1',
        base_quantity: '1',
        unit_price: '5000.00',
        price_rule: 'variant_price',
        tax_rate: '0',
        discount: '0.00',
        gross: '5000.00',
        base: '5000.00',
        tax: '0.00',
        total: '5000.00',
      },
    ],
    payments: [
      { method: 'card', amount: '30000.00' },
      { method: 'cash', amount: '30000.00' },
    ],
    totals: {
      subtotal: '52500.00',
      discounts: '2000.00',
      tax: '5845.00',
      total: '56345.00',
      taxes: [
        { rate: '0', base: '5000.00', tax: '0.00' },
        { rate: '0.05', base: '20000.00', tax: '1000.00' },
        { rate: '0.19', base: '25500.00', tax: '4845.00' },
      ],
    },
    paid: '60000.00',
    change: '3655.00',
    ...overrides,
  });
}

function renderReceipt(sale = aMixedSale()) {
  render(<Receipt sale={sale} business={business} location="Bodega principal" cashier="Ana Gómez" paper="80" />);
  return within(screen.getByRole('article', { name: `Tirilla de la venta N.º ${sale.number}` }));
}

// figure reads an amount of the receipt by its term ("Total", "Cambio").
function figure(receipt: ReturnType<typeof renderReceipt>, term: string) {
  return plain(receipt.getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
}

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

  it('muestra el IVA agrupado por tasa', () => {
    const receipt = renderReceipt();

    const table = within(receipt.getByRole('table', { name: 'IVA por tasa' }));
    const rows = table.getAllByRole('row').slice(1).map((row) => plain(row.textContent));
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

  it('muestra cada línea con su cantidad fraccionaria, su precio y su descuento', () => {
    const receipt = renderReceipt();

    const lines = receipt.getAllByRole('listitem');
    expect(lines).toHaveLength(4);
    const text = (name: string) => plain(receipt.getByRole('listitem', { name }).textContent);
    expect(text('Cable eléctrico 12 AWG')).toBe('Cable eléctrico 12 AWG2,5 × $ 3.000$ 7.500');
    expect(text('Martillo de uña 16 oz')).toBe('Martillo de uña 16 oz1 × $ 20.000$ 20.000Descuento-$ 2.000');
  });

  it('muestra la etiqueta de la variante junto al nombre del producto', () => {
    const receipt = renderReceipt(
      aSale({
        lines: [
          {
            variant_id: '01920000-0000-7000-8000-000000000121',
            product_id: '01920000-0000-7000-8000-000000000120',
            product_name: 'Camiseta básica',
            variant_label: 'Rojo · M',
            quantity: '1',
            unit_factor: '1',
            base_quantity: '1',
            unit_price: '10000.00',
            price_rule: 'variant_price',
            tax_rate: '0.19',
            discount: '0.00',
            gross: '10000.00',
            base: '10000.00',
            tax: '1900.00',
            total: '11900.00',
          },
        ],
      }),
    );

    const item = receipt.getByRole('listitem', { name: 'Camiseta básica (Rojo · M)' });
    expect(plain(item.textContent)).toBe('Camiseta básica (Rojo · M)1 × $ 10.000$ 10.000');
  });

  it.each([
    ['Cable eléctrico 12 AWG', '0.5', 'rollo', '100', '50', '180000.00', '90000.00', '0,5 rollo × $ 180.000$ 90.000'],
    ['Medias tobilleras', '2', 'paq', '3', '6', '70000.00', '140000.00', '2 paq × $ 70.000$ 140.000'],
  ])('muestra la unidad alternativa en que se vendió la línea: %s', (name, quantity, unit, factor, baseQuantity, price, gross, expected) => {
    const receipt = renderReceipt(
      aSale({
        lines: [
          {
            variant_id: '01920000-0000-7000-8000-000000000131',
            product_id: '01920000-0000-7000-8000-000000000130',
            product_name: name,
            quantity,
            unit_code: unit,
            unit_factor: factor,
            base_quantity: baseQuantity,
            unit_price: price,
            price_rule: 'variant_price',
            tax_rate: '0',
            discount: '0.00',
            gross,
            base: gross,
            tax: '0.00',
            total: gross,
          },
        ],
      }),
    );

    expect(plain(receipt.getByRole('listitem', { name }).textContent)).toBe(`${name}${expected}`);
  });

  it('muestra cada medio de pago, lo recibido y el cambio', () => {
    const receipt = renderReceipt();

    expect(figure(receipt, 'Tarjeta')).toBe('$ 30.000');
    expect(figure(receipt, 'Efectivo')).toBe('$ 30.000');
    expect(figure(receipt, 'Recibido')).toBe('$ 60.000');
    expect(figure(receipt, 'Cambio')).toBe('$ 3.655');
  });

  it('aclara que no es una factura electrónica', () => {
    const receipt = renderReceipt();

    expect(receipt.getByText('Este documento no es una factura electrónica de venta.')).toBeInTheDocument();
  });

  it('marca como ANULADA una venta anulada', () => {
    const receipt = renderReceipt(
      aMixedSale({
        status: 'voided',
        void: { user_id: '01920000-0000-7000-8000-000000000001', reason: 'Error', voided_at: '2026-10-11T16:00:00Z' },
      }),
    );

    expect(receipt.getByText('ANULADA')).toBeInTheDocument();
  });

  it('una venta confirmada no dice ANULADA', () => {
    const receipt = renderReceipt();

    expect(receipt.queryByText('ANULADA')).not.toBeInTheDocument();
  });
});
