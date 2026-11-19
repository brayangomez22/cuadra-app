import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { EmptyState, IconBox } from '@/shared/ui';
import { useIdleStockReport } from '../../api/reports';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { COST_PERMISSION } from '../../model/costs';
import { IdleFilters } from './components/IdleFilters/IdleFilters';
import { IdleResult } from './components/IdleResult/IdleResult';
import { IDLE_PAGE_SIZE, useIdleParams } from './hooks/useIdleParams';

/** /reportes/sin-movimiento: the stock that has not sold in N days, and the money it holds still. */
export function IdleStockPage() {
  const params = useReportParams();
  const idle = useIdleParams();
  const seesCosts = useCan()(COST_PERMISSION);
  const filters = {
    days: Number(idle.days),
    location_id: params.query.location_id,
    category_id: idle.category || undefined,
  };
  const report = useIdleStockReport({ ...filters, limit: IDLE_PAGE_SIZE, offset: idle.offset });

  return (
    <ReportSection
      title="Sin movimiento"
      description="Lo que tiene en existencias y no se vende: plata quieta que podría rematar o dejar de pedir."
      actions={<ExportButtons query={{ report: 'idle_stock', ...filters }} />}
    >
      <ReportFilters params={params} period={false}>
        <IdleFilters params={idle} />
      </ReportFilters>
      <QueryState
        queries={[report]}
        errorTitle="No pudimos cargar lo que no se vende"
        isEmpty={report.data?.total === 0}
        empty={
          <EmptyState
            icon={<IconBox />}
            title="Todo se está moviendo"
            description="Nada con existencias lleva tanto sin venderse."
          />
        }
      >
        {() => report.data && <IdleResult report={report.data} params={idle} seesCosts={seesCosts} />}
      </QueryState>
    </ReportSection>
  );
}
