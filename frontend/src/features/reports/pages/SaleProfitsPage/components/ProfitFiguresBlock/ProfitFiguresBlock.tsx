import { formatMoney, formatMargin } from '@/shared/format';
import { Figure, Figures } from '@/shared/ui';
import type { ProfitFigures } from '../../../../api/types';

/** What the period's sales brought in, what they cost when sold and what they left. */
export function ProfitFiguresBlock({ totals }: { totals: ProfitFigures }) {
  return (
    <Figures label="Utilidad del periodo">
      <Figure term="Ingreso sin IVA">{formatMoney(totals.revenue)}</Figure>
      <Figure term="Costo">{formatMoney(totals.cost ?? '0')}</Figure>
      <Figure term="Utilidad">{formatMoney(totals.profit ?? '0')}</Figure>
      <Figure term="Margen">{totals.margin ? formatMargin(totals.margin, 'rate') : '—'}</Figure>
    </Figures>
  );
}
