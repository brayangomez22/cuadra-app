import type { components } from '@/shared/api/schema';
import { query } from './requests';

type Customer = components['schemas']['Customer'];
type CustomerInput = components['schemas']['CustomerInput'];
type Segment = components['schemas']['CustomerSegment'];

export const customerIds = {
  builder: '01920000-0000-7000-8000-000000000501',
  boutique: '01920000-0000-7000-8000-000000000502',
  created: '01920000-0000-7000-8000-000000000503',
};

export const segmentIds = {
  builders: '01920000-0000-7000-8000-000000000601',
  wholesale: '01920000-0000-7000-8000-000000000602',
};

// Segments of two kinds of business: a hardware store's builders and a
// clothing store's wholesale buyers.
export const segments: Segment[] = [
  { id: segmentIds.builders, name: 'Maestro de obra' },
  { id: segmentIds.wholesale, name: 'Mayorista' },
];

/** A person who buys at a hardware store, in its builders segment. */
export function aBuilder(overrides: Partial<Customer> = {}): Customer {
  return {
    id: customerIds.builder,
    person_type: 'natural',
    document_type: 'cc',
    document_number: '1020304050',
    name: 'Carlos Pérez',
    phone: '3001234567',
    segment_id: segmentIds.builders,
    tax_responsibilities: ['R-99-PN'],
    active: true,
    created_at: '2026-10-01T15:00:00Z',
    ...overrides,
  };
}

/** A company that buys wholesale at a clothing store. */
export function aBoutique(overrides: Partial<Customer> = {}): Customer {
  return {
    id: customerIds.boutique,
    person_type: 'legal',
    document_type: 'nit',
    document_number: '901234567-7',
    name: 'Confecciones Sol SAS',
    trade_name: 'Boutique Sol',
    email: 'compras@boutiquesol.co',
    city: 'Medellín',
    segment_id: segmentIds.wholesale,
    tax_responsibilities: ['O-13', 'O-15'],
    active: true,
    created_at: '2026-10-02T15:00:00Z',
    ...overrides,
  };
}

// The customers answer like the API: every word of q in the name or trade
// name, or the document number as a prefix; and the segment's filter.
export function searchCustomers(customers: Customer[], request: Request) {
  const params = query(request);
  const words = (params.get('q') ?? '').toLowerCase().split(/\s+/).filter(Boolean);
  const segment = params.get('segment_id');
  return customers.filter(
    (c) =>
      words.every(
        (w) => `${c.name} ${c.trade_name ?? ''}`.toLowerCase().includes(w) || c.document_number.startsWith(w),
      ) &&
      (!segment || c.segment_id === segment),
  );
}

/** The body that saves a customer as it is: everything but what the API assigns. */
export function customerInput(customer: Customer): CustomerInput {
  const input: Partial<Customer> = { ...customer };
  delete input.id;
  delete input.created_at;
  return input as CustomerInput;
}
