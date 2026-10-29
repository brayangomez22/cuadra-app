import { useCan } from '@/features/auth';
import type { ChosenCustomer } from '@/features/customers';
import { useCustomerCredit } from '@/features/receivables';
import { isZero } from '@/shared/format';

/** What the sale's customer can buy on credit: undefined without a customer with a credit limit. */
export type SaleCredit = { available: string } | undefined;

/**
 * The credit of the sale's customer, for who may read it. A customer with a
 * limit is offered credit even with nothing available: going over it asks
 * for an administrator's authorization.
 */
export function useSaleCredit(customer: ChosenCustomer | null): SaleCredit {
  const canRead = useCan()('receivables:read');
  const credit = useCustomerCredit(customer?.id, { enabled: canRead });
  const data = credit.data;
  if (!customer || data?.customer_id !== customer.id || isZero(data.credit_limit)) return undefined;
  return { available: data.available };
}
