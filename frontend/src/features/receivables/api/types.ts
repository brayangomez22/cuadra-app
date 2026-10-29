import type { components, operations } from '@/shared/api/schema';

export type CustomerCredit = components['schemas']['CustomerCredit'];
export type CreditTermsInput = components['schemas']['CreditTermsInput'];
export type Receivable = components['schemas']['Receivable'];
export type CreditMovement = components['schemas']['CreditMovement'];
export type CustomerPaymentMethod = components['schemas']['CustomerPaymentMethod'];
export type CustomerPaymentInput = components['schemas']['CustomerPaymentInput'];
export type CustomerPayment = components['schemas']['CustomerPayment'];
export type AgingBuckets = components['schemas']['AgingBuckets'];
export type AgingRow = components['schemas']['AgingRow'];
export type PageQuery = NonNullable<operations['getReceivablesAging']['parameters']['query']>;
