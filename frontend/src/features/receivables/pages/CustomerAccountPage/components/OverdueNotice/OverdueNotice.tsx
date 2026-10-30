import { formatMoney, todayInBogota } from '@/shared/format';
import { Alert } from '@/shared/ui';
import type { Receivable } from '../../../../api/types';
import { overdueSummary } from '../../../../model/overdue';

type Props = { receivables: Receivable[] };

const plural = (n: number, one: string, many: string) => `${String(n)} ${n === 1 ? one : many}`;

/** Whether the customer is late, before the figures: how much is past due and since when. */
export function OverdueNotice({ receivables }: Props) {
  const overdue = overdueSummary(receivables, todayInBogota());
  if (!overdue) return null;
  return (
    <Alert tone="warning" title="En mora">
      {formatMoney(overdue.balance)} vencido en {plural(overdue.count, 'venta', 'ventas')}; la más antigua lleva{' '}
      {plural(overdue.oldestDays, 'día', 'días')}.
    </Alert>
  );
}
