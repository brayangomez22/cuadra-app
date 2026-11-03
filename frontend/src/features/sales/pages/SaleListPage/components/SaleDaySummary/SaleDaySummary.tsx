import { formatMoney } from '@/shared/format';
import { Figure, Figures } from '@/shared/ui';
import type { SalePage } from '../../../../api/types';

/** The day at a glance, over all its pages: how many sales, what the confirmed ones add up to and how many were voided. */
export function SaleDaySummary({ page }: { page: SalePage }) {
  return (
    <Figures label="Resumen del día">
      <Figure term="Ventas">{page.total}</Figure>
      <Figure term="Vendido">{formatMoney(page.sold)}</Figure>
      <Figure term="Anuladas" tone={page.voided > 0 ? 'warning' : undefined}>
        {page.voided}
      </Figure>
    </Figures>
  );
}
