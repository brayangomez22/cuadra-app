import { formatDay } from '@/shared/format';
import { Badge } from '@/shared/ui';
import { dueState } from '../../../../model/dueState';

type Props = { dueDate: string; today: string };

const days = (n: number) => (n === 1 ? '1 día' : `${String(n)} días`);

/** When a receivable is due, and whether it is overdue (in words, not only in color). */
export function DueDate({ dueDate, today }: Props) {
  const { state, days: count } = dueState(dueDate, today);
  if (state === 'overdue') return <Badge tone="danger">Vencida hace {days(count)}</Badge>;
  if (state === 'due_today') return <Badge tone="warning">Vence hoy</Badge>;
  return <>Vence el {formatDay(dueDate)}</>;
}
