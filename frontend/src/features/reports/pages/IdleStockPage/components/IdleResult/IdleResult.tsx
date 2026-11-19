import { formatMoney } from '@/shared/format';
import { Figure, Figures, Pagination } from '@/shared/ui';
import type { IdleStockReport } from '../../../../api/types';
import { IDLE_PAGE_SIZE, type IdleParams } from '../../hooks/useIdleParams';
import { IdleTable } from '../IdleTable/IdleTable';

type Props = { report: IdleStockReport; params: IdleParams; seesCosts: boolean };

/** How much is idle, the products and their pages. */
export function IdleResult({ report, params, seesCosts }: Props) {
  return (
    <>
      <Figures label="Plata quieta">
        <Figure term="Productos">{report.total}</Figure>
        {seesCosts && <Figure term="Valor al costo">{formatMoney(report.value ?? '0')}</Figure>}
      </Figures>
      <IdleTable rows={report.items} seesCosts={seesCosts} />
      <Pagination
        label="Páginas de productos"
        total={report.total}
        limit={IDLE_PAGE_SIZE}
        offset={params.offset}
        onOffsetChange={params.setOffset}
      />
    </>
  );
}
