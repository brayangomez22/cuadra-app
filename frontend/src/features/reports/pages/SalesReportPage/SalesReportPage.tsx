import { enumParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { EmptyState, Field, IconChart, Select } from '@/shared/ui';
import { useSalesReport } from '../../api/reports';
import type { SalesReportGroup } from '../../api/types';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { SalesFigures } from './components/SalesFigures/SalesFigures';
import { SalesTable } from './components/SalesTable/SalesTable';

const groups: { value: SalesReportGroup; label: string }[] = [
  { value: 'day', label: 'Día' },
  { value: 'user', label: 'Vendedor' },
  { value: 'payment_method', label: 'Medio de pago' },
  { value: 'category', label: 'Categoría' },
  { value: 'location', label: 'Sede' },
];
const groupParam = enumParam(
  groups.map((g) => g.value),
  'day',
);

/** /reportes/ventas: the period's confirmed sales by day, seller, payment method, category or location. */
export function SalesReportPage() {
  const params = useReportParams();
  const [group, setGroup] = useSearchParamsState('agrupar', groupParam);
  const query = { ...params.query, group_by: group };
  const report = useSalesReport(query, !params.problem);

  return (
    <ReportSection
      title="Ventas"
      description="Las ventas confirmadas del periodo; las anuladas no cuentan."
      actions={<ExportButtons query={{ report: 'sales', ...query }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params}>
        <Field label="Agrupar por">
          <Select
            options={groups}
            value={group}
            onValueChange={(value) => setGroup(groupParam.parse(value) ?? 'day')}
          />
        </Field>
      </ReportFilters>
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar las ventas"
          isEmpty={report.data?.totals.sales_count === 0}
          empty={
            <EmptyState
              icon={<IconChart />}
              title="Sin ventas en el periodo"
              description="Pruebe con otras fechas u otra sede."
            />
          }
        >
          {() =>
            report.data && (
              <>
                <SalesFigures totals={report.data.totals} />
                <SalesTable group={group} rows={report.data.items} />
              </>
            )
          }
        </QueryState>
      )}
    </ReportSection>
  );
}
