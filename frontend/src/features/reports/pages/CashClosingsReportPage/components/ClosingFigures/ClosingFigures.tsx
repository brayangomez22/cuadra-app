import { formatMoney, isZero } from '@/shared/format';
import { Figure, Figures } from '@/shared/ui';
import type { CashClosingsReport } from '../../../../api/types';

/** The period's closings: how many, what they sold and what was short or over. */
export function ClosingFigures({ totals }: { totals: CashClosingsReport['totals'] }) {
  return (
    <Figures label="Cierres del periodo">
      <Figure term="Cierres">{totals.closings}</Figure>
      <Figure term="Vendido">{formatMoney(totals.sales_total)}</Figure>
      <Figure term="Faltantes" tone={isZero(totals.shortage) ? undefined : 'danger'}>
        {formatMoney(totals.shortage)}
      </Figure>
      <Figure term="Sobrantes">{formatMoney(totals.overage)}</Figure>
    </Figures>
  );
}
