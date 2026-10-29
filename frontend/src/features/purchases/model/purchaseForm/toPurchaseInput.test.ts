import { aGarment, aProduct, boxOf12, ids, unitOf, variantId, withUnits } from '@/test/fixtures';
import type { PurchaseValues } from './purchaseSchema';
import { toPurchaseInput } from './toPurchaseInput';

const supplierId = '01920000-0000-7000-8000-000000000a01';
const tubes = withUnits(aProduct(), [boxOf12]);
const shirts = aGarment();

function values(overrides: Partial<PurchaseValues> = {}): PurchaseValues {
  return {
    supplier_id: supplierId,
    location_id: ids.main,
    invoice_number: ' FE 1234 ',
    issued_on: '2026-10-20',
    payment: 'credit',
    due_on: '2026-11-19',
    notes: '',
    lines: [
      {
        item: { product: tubes, variant: tubes.variants[0]! },
        unit_id: unitOf('caja').id,
        quantity: '2',
        unit_cost: '96000',
        tax_rate: '0.19',
      },
      {
        item: { product: shirts, variant: shirts.variants[1]! },
        unit_id: '',
        quantity: '6',
        unit_cost: '15000.50',
        tax_rate: '0',
      },
    ],
    ...overrides,
  };
}

describe('toPurchaseInput', () => {
  it('arma la compra a crédito con sus líneas: la unidad alternativa lleva unit_id y la base no', () => {
    expect(toPurchaseInput(values())).toEqual({
      supplier_id: supplierId,
      location_id: ids.main,
      invoice_number: 'FE 1234',
      issued_on: '2026-10-20',
      payment: 'credit',
      due_on: '2026-11-19',
      lines: [
        { variant_id: ids.pipe, unit_id: unitOf('caja').id, quantity: '2', unit_cost: '96000', tax_rate: '0.19' },
        { variant_id: variantId('S', 'Azul'), quantity: '6', unit_cost: '15000.50', tax_rate: '0' },
      ],
    });
  });

  it('de contado no envía el vencimiento', () => {
    expect(toPurchaseInput(values({ payment: 'cash' }))).not.toHaveProperty('due_on');
  });

  it('a crédito sin vencimiento deja que la API use el plazo del proveedor', () => {
    expect(toPurchaseInput(values({ due_on: '' }))).not.toHaveProperty('due_on');
  });

  it('envía las notas solo si tienen texto', () => {
    expect(toPurchaseInput(values({ notes: '  Llegó una caja golpeada  ' })).notes).toBe('Llegó una caja golpeada');
    expect(toPurchaseInput(values({ notes: '   ' }))).not.toHaveProperty('notes');
  });
});
