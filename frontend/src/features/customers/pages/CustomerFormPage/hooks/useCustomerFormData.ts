import { useCustomer } from '../../../api/customers';
import { useCustomerSegments } from '../../../api/segments';
import type { Customer, CustomerSegment } from '../../../api/types';

export type CustomerFormData = { customer?: Customer; segments: CustomerSegment[] };

/** What the form needs: the tenant's segments and, to edit, the customer. data waits for all of it. */
export function useCustomerFormData(customerId: string | undefined) {
  const customer = useCustomer(customerId);
  const segments = useCustomerSegments();
  const queries = customerId ? [customer, segments] : [segments];
  const ready = segments.data && (!customerId || customer.data);
  const data: CustomerFormData | undefined = ready ? { customer: customer.data, segments: segments.data } : undefined;
  return { queries, data };
}
