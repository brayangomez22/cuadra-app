import type { components } from '@/shared/api/schema';
import { fromUnits, toUnits } from '@/shared/money/decimal';
import { customerIds } from './customers';

type CustomerCredit = components['schemas']['CustomerCredit'];
type CreditMovement = components['schemas']['CreditMovement'];
type AgingBuckets = components['schemas']['AgingBuckets'];

const saleIds = {
  first: '01920000-0000-7000-8000-000000000701',
  second: '01920000-0000-7000-8000-000000000702',
};

/**
 * The credit of the hardware store's builder: 1.500.000 of limit at 30 days,
 * owing 350.000 in two sales, one overdue since 20/10/2026 and one due on
 * 25/11/2026.
 */
export function aCredit(overrides: Partial<CustomerCredit> = {}): CustomerCredit {
  return {
    customer_id: customerIds.builder,
    credit_limit: '1500000.00',
    term_days: 30,
    balance: '350000.00',
    available: '1150000.00',
    receivables: [
      {
        id: '01920000-0000-7000-8000-000000000711',
        sale_id: saleIds.first,
        amount: '200000.00',
        balance: '150000.00',
        due_date: '2026-10-20',
        status: 'open',
        created_at: '2026-09-20T15:00:00Z',
      },
      {
        id: '01920000-0000-7000-8000-000000000712',
        sale_id: saleIds.second,
        amount: '200000.00',
        balance: '200000.00',
        due_date: '2026-11-25',
        status: 'open',
        created_at: '2026-10-26T15:00:00Z',
      },
    ],
    ...overrides,
  };
}

/** The clothing store's wholesale buyer, who has no credit: its limit is 0. */
export function noCredit(customerId = customerIds.boutique): CustomerCredit {
  return {
    customer_id: customerId,
    credit_limit: '0.00',
    term_days: 0,
    balance: '0.00',
    available: '0.00',
    receivables: [],
  };
}

/** The builder's statement, most recent first: two credit sales and a payment in cash. */
export const statement: CreditMovement[] = [
  {
    kind: 'charge',
    at: '2026-10-26T15:00:00Z',
    amount: '200000.00',
    balance: '350000.00',
    sale_id: saleIds.second,
    due_date: '2026-11-25',
  },
  { kind: 'payment', at: '2026-10-05T15:00:00Z', amount: '50000.00', balance: '150000.00', method: 'cash' },
  {
    kind: 'charge',
    at: '2026-09-20T15:00:00Z',
    amount: '200000.00',
    balance: '200000.00',
    sale_id: saleIds.first,
    due_date: '2026-10-20',
  },
];

/** Aging buckets: amounts by bucket, the rest at 0. overdue and total are computed. */
export function buckets(amounts: Partial<Omit<AgingBuckets, 'overdue' | 'total'>>): AgingBuckets {
  const all = {
    current: '0.00',
    days_1_30: '0.00',
    days_31_60: '0.00',
    days_61_90: '0.00',
    over_90: '0.00',
    ...amounts,
  };
  const sum = (values: string[]) => fromUnits(values.reduce((a, v) => a + toUnits(v), 0n));
  const overdue = sum([all.days_1_30, all.days_31_60, all.days_61_90, all.over_90]);
  return { ...all, overdue, total: sum([all.current, overdue]) };
}
