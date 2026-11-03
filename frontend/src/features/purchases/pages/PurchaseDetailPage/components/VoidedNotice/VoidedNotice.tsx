import { formatDateTime } from '@/shared/format';
import { Alert } from '@/shared/ui';
import type { Purchase } from '../../../../api/types';

/** When a voided purchase was voided and why. */
export function VoidedNotice({ purchase }: { purchase: Purchase }) {
  if (!purchase.voided_at) return null;
  return (
    <Alert tone="warning">{`Anulada el ${formatDateTime(purchase.voided_at)} · Motivo: ${purchase.void_reason ?? ''}`}</Alert>
  );
}
