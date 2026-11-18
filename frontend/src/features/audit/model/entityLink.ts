import { paths } from '@/app/routes/paths';
import type { AuditEvent } from '../api/types';

/** Where to see the entity of an event, when the app has a screen for it by its id. */
export function entityLink(event: AuditEvent): { to: string; label: string } | null {
  const id = event.entity_id;
  if (!id) return null;
  switch (event.entity_type) {
    case 'sale':
      return { to: paths.sales.detail(id), label: 'Ver venta' };
    case 'purchase':
      return { to: paths.purchases.detail(id), label: 'Ver compra' };
    case 'cash_session':
      return { to: paths.sales.cashReport(id), label: 'Ver caja' };
    case 'customer':
      return { to: paths.receivables.account(id), label: 'Ver cuenta' };
    default:
      return null;
  }
}
