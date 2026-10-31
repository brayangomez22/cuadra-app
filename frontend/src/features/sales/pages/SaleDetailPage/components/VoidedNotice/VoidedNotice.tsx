import { formatDateTime } from '@/shared/format';
import { Alert } from '@/shared/ui';
import type { Sale } from '../../../../api/types';

/** When a voided sale was voided and why. */
export function VoidedNotice({ info }: { info: NonNullable<Sale['void']> }) {
  return <Alert tone="warning">{`Anulada el ${formatDateTime(info.voided_at)} · Motivo: ${info.reason}`}</Alert>;
}
