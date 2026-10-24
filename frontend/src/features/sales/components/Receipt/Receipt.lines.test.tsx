import { plain } from '@/test/fixtures';
import { aSale } from '../../testing/fixtures';
import { renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
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

  // Each row is an object: eight positional arguments would not read.
  it.each([
    {
      name: 'Cable eléctrico 12 AWG',
      quantity: '0.5',
      unit: 'rollo',
      factor: '100',
      baseQuantity: '50',
      price: '180000.00',
      gross: '90000.00',
      expected: '0,5 rollo × $ 180.000$ 90.000',
    },
    {
      name: 'Medias tobilleras',
      quantity: '2',
      unit: 'paq',
      factor: '3',
      baseQuantity: '6',
      price: '70000.00',
      gross: '140000.00',
      expected: '2 paq × $ 70.000$ 140.000',
    },
  ])(
    'muestra la unidad alternativa en que se vendió la línea: $name',
    ({ name, quantity, unit, factor, baseQuantity, price, gross, expected }) => {
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
    },
  );
});
