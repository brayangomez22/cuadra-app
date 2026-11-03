import { formatMoney, isZero } from '@/shared/format';
import { Figure, Figures } from '@/shared/ui';
import type { PayablesTotals } from '../../../../api/types';

/** The totals of what is owed: all of it, what is overdue (in red only when there is some) and what falls due within a week. */
export function PayablesSummary({ totals }: { totals: PayablesTotals }) {
  return (
    <Figures label="Resumen de lo que se debe">
      <Figure term="Se debe en total">{formatMoney(totals.balance)}</Figure>
      <Figure term="Vencido" tone={isZero(totals.overdue) ? undefined : 'danger'}>
        {formatMoney(totals.overdue)}
      </Figure>
      <Figure term="Vence en 7 días">{formatMoney(totals.due_soon)}</Figure>
    </Figures>
  );
}
