import { aGarment, aProduct, ids } from '@/test/fixtures';
import { aDistributor, aTextileSupplier, supplierIds } from '../../testing/fixtures';
import type { PurchaseValues } from './purchaseSchema';
import { purchaseReview } from './purchaseReview';

const tubes = aProduct();
const shirts = aGarment();
const data = {
  suppliers: [aDistributor(), aTextileSupplier()],
  locations: [{ value: ids.main, label: 'Bodega principal' }],
};

function values(overrides: Partial<PurchaseValues> = {}): PurchaseValues {
  return {
    supplier_id: supplierIds.hardware,
    location_id: ids.main,
    invoice_number: ' FE 1234 ',
    issued_on: '2026-10-20',
    payment: 'cash',
    due_on: '',
    notes: '',
    lines: [
      {
        item: { product: tubes, variant: tubes.variants[0]! },
        unit_id: '',
        quantity: '10',
        unit_cost: '2500',
        tax_rate: '0.19',
      },
    ],
    ...overrides,
  };
}

describe('purchaseReview', () => {
  it('resume una compra de contado de ferretería: proveedor, sede, factura, líneas y total con IVA', () => {
    expect(purchaseReview(values(), data)).toEqual({
      supplier: 'Distribuidora Andina SAS',
      location: 'Bodega principal',
      invoice: 'FE 1234',
      lines: 1,
      total: '29750.00',
      dueOn: null,
    });
  });

  it('resume una compra a crédito de ropa con su vencimiento y varias líneas', () => {
    const line = (variant: number) => ({
      item: { product: shirts, variant: shirts.variants[variant]! },
      unit_id: '',
      quantity: '6',
      unit_cost: '15000',
      tax_rate: '0.19',
    });
    const review = purchaseReview(
      values({ supplier_id: supplierIds.textiles, payment: 'credit', due_on: '2026-11-19', lines: [line(0), line(1)] }),
      data,
    );

    expect(review).toMatchObject({
      supplier: 'Confecciones del Valle',
      lines: 2,
      total: '214200.00',
      dueOn: '2026-11-19',
    });
  });

  it('a crédito sin vencimiento escrito lo deja para el plazo del proveedor', () => {
    expect(purchaseReview(values({ payment: 'credit', due_on: '' }), data).dueOn).toBe('');
  });
});
