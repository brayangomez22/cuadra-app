import { plain } from '@/test/fixtures';
import { aSale } from '../../testing/fixtures';
import { renderReceipt } from '../../testing/receipt';

// A line sold in the product's base unit, as the sale froze it.
function baseUnitLine(name: string, quantity: string, baseUnit?: string) {
  return {
    variant_id: '01920000-0000-7000-8000-000000000141',
    product_id: '01920000-0000-7000-8000-000000000140',
    product_name: name,
    quantity,
    unit_factor: '1',
    base_quantity: quantity,
    ...(baseUnit && { base_unit_code: baseUnit }),
    unit_price: '2000.00',
    price_rule: 'variant_price',
    tax_rate: '0',
    tax_treatment: 'excluded' as const,
    discount: '0.00',
    gross: '4000.00',
    base: '4000.00',
    tax: '0.00',
    total: '4000.00',
  };
}

describe('Receipt: unidad base', () => {
  it.each([
    { negocio: 'ferretería', name: 'Cable eléctrico 12 AWG', unit: 'm', expected: '2 m × $ 2.000$ 4.000' },
    { negocio: 'almacén de calzado', name: 'Medias deportivas', unit: 'par', expected: '2 par × $ 2.000$ 4.000' },
  ])('muestra la unidad base de la línea ($negocio)', ({ name, unit, expected }) => {
    const receipt = renderReceipt(aSale({ lines: [baseUnitLine(name, '2', unit)] }));

    expect(plain(receipt.getByRole('listitem', { name }).textContent)).toBe(`${name}${expected}`);
  });

  it('una venta guardada antes de congelar la unidad se ve como antes', () => {
    const receipt = renderReceipt(aSale({ lines: [baseUnitLine('Cuaderno', '2')] }));

    expect(plain(receipt.getByRole('listitem', { name: 'Cuaderno' }).textContent)).toBe('Cuaderno2 × $ 2.000$ 4.000');
  });
});
