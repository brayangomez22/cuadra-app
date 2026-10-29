import { aCement, aProduct } from '@/test/fixtures';
import { emptyLine, purchaseDefaults } from './purchaseDefaults';
import { purchaseSchema } from './purchaseSchema';

const cement = aCement();
const filled = {
  ...purchaseDefaults('2026-10-28'),
  supplier_id: 'proveedor',
  location_id: 'sede',
  invoice_number: 'FE 1234',
  lines: [
    {
      ...emptyLine(),
      item: { product: cement, variant: cement.variants[0]! },
      quantity: '10',
      unit_cost: '28000',
      tax_rate: '0.19',
    },
  ],
};

describe('purchaseDefaults', () => {
  it('arranca de contado, con la fecha de hoy y una línea vacía', () => {
    expect(purchaseDefaults('2026-10-28')).toMatchObject({ payment: 'cash', issued_on: '2026-10-28', due_on: '' });
    expect(purchaseDefaults('2026-10-28').lines).toEqual([emptyLine()]);
  });
});

describe('purchaseSchema', () => {
  it('acepta una compra completa y lee la cantidad con coma decimal', () => {
    const line = { ...filled.lines[0]!, quantity: '2,5' };
    expect(purchaseSchema.parse({ ...filled, lines: [line] }).lines[0]?.quantity).toBe('2.5');
  });

  it.each([
    { name: 'exige el proveedor', fields: { supplier_id: '' }, path: 'supplier_id', message: 'Elija el proveedor' },
    { name: 'exige la sede', fields: { location_id: '' }, path: 'location_id', message: 'Elija la sede' },
    {
      name: 'exige el número de la factura',
      fields: { invoice_number: ' ' },
      path: 'invoice_number',
      message: 'Escriba el número de la factura',
    },
    {
      name: 'exige la fecha de la factura',
      fields: { issued_on: '' },
      path: 'issued_on',
      message: 'Escriba la fecha de la factura',
    },
  ])('$name', ({ fields, path, message }) => {
    const issue = purchaseSchema.safeParse({ ...filled, ...fields }).error?.issues[0];
    expect(issue?.path).toEqual([path]);
    expect(issue?.message).toBe(message);
  });

  it('marca en su fila la línea sin producto o con cantidad en cero', () => {
    const tubes = aProduct();
    const lines = [
      filled.lines[0]!,
      { ...emptyLine(), quantity: '3' },
      {
        ...emptyLine(),
        item: { product: tubes, variant: tubes.variants[0]! },
        quantity: '0',
        unit_cost: '8000',
        tax_rate: '0.19',
      },
    ];
    const issues = purchaseSchema.safeParse({ ...filled, lines }).error?.issues ?? [];
    expect(issues.map((i) => [i.path.join('.'), i.message])).toEqual(
      expect.arrayContaining([
        ['lines.1.item', 'Elija el producto'],
        ['lines.2.quantity', 'La cantidad debe ser mayor que cero'],
      ]),
    );
  });
});
