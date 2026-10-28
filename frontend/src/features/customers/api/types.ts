import type { components, operations } from '@/shared/api/schema';

export type Customer = components['schemas']['Customer'];
export type CustomerInput = components['schemas']['CustomerInput'];
export type CustomerSegment = components['schemas']['CustomerSegment'];
export type PersonType = components['schemas']['PersonType'];
export type DocumentType = components['schemas']['DocumentType'];
export type TaxResponsibility = components['schemas']['TaxResponsibility'];
export type CustomerQuery = NonNullable<operations['listCustomers']['parameters']['query']>;
export type CustomerStatus = NonNullable<CustomerQuery['status']>;

/** The customer a sale goes to, as the POS shows it and keeps it while the sale is open. */
export type ChosenCustomer = { id: string; name: string; document: string };
