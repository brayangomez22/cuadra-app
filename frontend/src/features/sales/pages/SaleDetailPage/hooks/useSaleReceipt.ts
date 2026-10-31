import { useState } from 'react';
import { useAuth, type Me } from '@/features/auth';
import { documentLabel, useCustomer } from '@/features/customers';
import { useLocations } from '@/features/inventory';
import { useSale } from '../../../api/sales';
import type { Paper } from '../../../model/paper';
import { loadPaper, savePaper } from '../../../model/storage';

/**
 * A sale and what its receipt names: the business, the location, the customer
 * and the cashier when known (users have no endpoint yet, so only for one's
 * own sale). A sale with a customer waits for it: without its name the receipt
 * would read as a final consumer.
 */
export function useSaleReceipt(saleId: string) {
  const sale = useSale(saleId);
  const customerId = sale.data?.customer_id;
  const customer = useCustomer(customerId);
  const locations = useLocations();
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const [paper, setPaper] = useState(loadPaper);

  const data = sale.data;
  const receipt = data && {
    sale: data,
    business: businessOf(user),
    location: locationName(locations.data, data.location_id),
    cashier: user?.id === data.user_id ? user.name : undefined,
    customer: customer.data && { name: customer.data.name, document: documentLabel(customer.data) },
    paper,
  };
  return {
    queries: customerId ? [sale, customer] : [sale],
    receipt,
    choosePaper: (next: Paper) => {
      setPaper(next);
      savePaper(next);
    },
  };
}

function businessOf(user: Me | undefined) {
  return { name: user?.tenant_name ?? '', nit: user?.tenant_nit ?? '' };
}

function locationName(locations: { id: string; name: string }[] | undefined, id: string): string {
  return locations?.find((l) => l.id === id)?.name ?? '';
}
