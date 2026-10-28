import { useCan } from '@/features/auth';
import { CustomerPicker } from '@/features/customers';
import type { SaleCustomer as SaleCustomerState } from '../../hooks/useSaleCustomer';

type Props = { sale: SaleCustomerState; onDone: () => void };

/** The customer of the sale, for who may look customers up; without it every sale is to a final consumer. */
export function SaleCustomer({ sale, onDone }: Props) {
  const canRead = useCan()('customers:read');
  if (!canRead) return null;
  return <CustomerPicker value={sale.customer} onChange={sale.setCustomer} onDone={onDone} />;
}
