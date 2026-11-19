import { useCan } from '@/features/auth';
import { enumParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { formatMoney } from '@/shared/format';
import { QueryState } from '@/shared/query';
import { EmptyState, Field, Figure, Figures, IconClipboardCheck, Select } from '@/shared/ui';
import { useStockAdjustmentsReport } from '../../api/reports';
import type { StockAdjustmentsReportGroup } from '../../api/types';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { COST_PERMISSION } from '../../model/costs';
import { AdjustmentsTable } from './components/AdjustmentsTable/AdjustmentsTable';

const groups: { value: StockAdjustmentsReportGroup; label: string }[] = [
  { value: 'user', label: 'Usuario' },
  { value: 'variant', label: 'Producto' },
];
const groupParam = enumParam<StockAdjustmentsReportGroup>(['user', 'variant'], 'user');

/** /reportes/ajustes: the stock adjustments (manual and from counts) by user or product, valued at cost. */
export function StockAdjustmentsPage() {
  const params = useReportParams();
  const [group, setGroup] = useSearchParamsState('agrupar', groupParam);
  const seesCosts = useCan()(COST_PERMISSION);
  const query = { ...params.query, group_by: group };
  const report = useStockAdjustmentsReport(query, !params.problem);

  return (
    <ReportSection
      title="Ajustes de inventario"
      description="Lo que se sumó o restó del inventario a mano o al aprobar una toma física, y lo que valía."
      actions={<ExportButtons query={{ report: 'stock_adjustments', ...query }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params}>
        <Field label="Agrupar por">
          <Select options={groups} value={group} onValueChange={(v) => setGroup(groupParam.parse(v) ?? 'user')} />
        </Field>
      </ReportFilters>
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar los ajustes"
          isEmpty={report.data?.adjustments === 0}
          empty={
            <EmptyState
              icon={<IconClipboardCheck />}
              title="Sin ajustes en el periodo"
              description="Pruebe con otras fechas."
            />
          }
        >
          {() => (
            <>
              <Figures label="Ajustes del periodo">
                <Figure term="Ajustes">{report.data?.adjustments}</Figure>
                {seesCosts && <Figure term="Valor al costo">{formatMoney(report.data?.value ?? '0')}</Figure>}
              </Figures>
              <AdjustmentsTable group={group} rows={report.data?.items ?? []} seesCosts={seesCosts} />
            </>
          )}
        </QueryState>
      )}
    </ReportSection>
  );
}
