import { render, screen, within } from '@testing-library/react';
import { plain } from '@/test/fixtures';
import { Receipt } from '../components/Receipt/Receipt';
import { aSale, type Sale } from './fixtures';

const business = { name: 'Ferretería El Tornillo', nit: '900123456-8' };

// Two lines at 19 % (one with a discount, one fractional), one at 5 % and one at 0 %.
const mixedLines: Sale['lines'] = [
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
];

// aMixedSale is a sale with IVA at three rates; two lines at 19 % add up in one group.
export function aMixedSale(overrides: Partial<Sale> = {}): Sale {
  return aSale({
    number: 42,
    lines: mixedLines,
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

/** customer is who bought, as the POS chose it; without one, the sale is to a final consumer. */
export function renderReceipt(sale = aMixedSale(), customer?: { name: string; document: string }) {
  render(
    <Receipt
      sale={sale}
      business={business}
      location="Bodega principal"
      cashier="Ana Gómez"
      customer={customer}
      paper="80"
    />,
  );
  return within(screen.getByRole('article', { name: `Tirilla de la venta N.º ${sale.number}` }));
}

// figure reads an amount of the receipt by its term ("Total", "Cambio").
export function figure(receipt: ReturnType<typeof renderReceipt>, term: string) {
  return plain(receipt.getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
}
