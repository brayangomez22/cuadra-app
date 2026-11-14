import { useCustomer } from '../../../api/customers';
import { useMunicipalities } from '../../../api/municipalities';
import { useCustomerSegments } from '../../../api/segments';
import type { Customer, CustomerSegment, Municipality } from '../../../api/types';

export type CustomerFormData = { customer?: Customer; segments: CustomerSegment[]; municipalities: Municipality[] };

/** What the form needs: the tenant's segments, the DANE's municipalities and, to edit, the customer. data waits for all of it. */
export function useCustomerFormData(customerId: string | undefined) {
  const customer = useCustomer(customerId);
  const segments = useCustomerSegments();
  const municipalities = useMunicipalities();
  const shared = [segments, municipalities];
  const queries = customerId ? [customer, ...shared] : shared;
  const ready = segments.data && municipalities.data && (!customerId || customer.data);
  const data: CustomerFormData | undefined = ready
    ? { customer: customer.data, segments: segments.data, municipalities: municipalities.data }
    : undefined;
  return { queries, data };
}
