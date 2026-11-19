import { formatMoney } from '@/shared/format';
import { Figure, Figures } from '@/shared/ui';
import type { SalesReport } from '../../../../api/types';

/** How many sales the period had and what they added up to, VAT apart. */
export function SalesFigures({ totals }: { totals: SalesReport['totals'] }) {
  return (
    <Figures label="Totales del periodo">
      <Figure term="Ventas">{totals.sales_count}</Figure>
      <Figure term="Sin IVA">{formatMoney(totals.net)}</Figure>
      <Figure term="IVA">{formatMoney(totals.tax)}</Figure>
      <Figure term="Total">{formatMoney(totals.total)}</Figure>
    </Figures>
  );
}
