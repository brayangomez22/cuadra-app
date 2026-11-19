import { pageParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { EmptyState, IconReceipt, Pagination } from '@/shared/ui';
import { useSaleProfitsReport } from '../../api/reports';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { ProfitFiguresBlock } from './components/ProfitFiguresBlock/ProfitFiguresBlock';
import { ProfitsTable } from './components/ProfitsTable/ProfitsTable';

const PAGE_SIZE = 50;

/** /reportes/utilidad: what each sale left, at the cost of its moment (only for whoever sees costs). */
export function SaleProfitsPage() {
  const params = useReportParams();
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAGE_SIZE;
  const report = useSaleProfitsReport({ ...params.query, limit: PAGE_SIZE, offset }, !params.problem);

  return (
    <ReportSection
      title="Utilidad por venta"
      description="El ingreso sin IVA de cada venta menos el costo promedio de lo vendido en ese momento."
      actions={<ExportButtons query={{ report: 'sale_profits', ...params.query }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params} />
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar la utilidad"
          isEmpty={report.data?.total === 0}
          empty={
            <EmptyState
              icon={<IconReceipt />}
              title="Sin ventas en el periodo"
              description="Pruebe con otras fechas."
            />
          }
        >
          {() =>
            report.data && (
              <>
                <ProfitFiguresBlock totals={report.data.totals} />
                <ProfitsTable rows={report.data.items} />
                <Pagination
                  label="Páginas de ventas"
                  total={report.data.total}
                  limit={PAGE_SIZE}
                  offset={offset}
                  onOffsetChange={(next) => setPage(next / PAGE_SIZE + 1)}
                />
              </>
            )
          }
        </QueryState>
      )}
    </ReportSection>
  );
}
