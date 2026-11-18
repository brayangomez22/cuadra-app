import { formatDateTime } from '@/shared/format';
import { Alert } from '@/shared/ui';
import type { Sale } from '../../../../api/types';

/** When a voided sale was voided, by whom (when the user still exists) and why. */
export function VoidedNotice({ info }: { info: NonNullable<Sale['void']> }) {
  const by = info.user_name ? ` por ${info.user_name}` : '';
  return <Alert tone="warning">{`Anulada el ${formatDateTime(info.voided_at)}${by} · Motivo: ${info.reason}`}</Alert>;
}
