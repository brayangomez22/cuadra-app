import { enumParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { EmptyState, Field, IconCashStack, Select } from '@/shared/ui';
import { useCashClosingsReport } from '../../api/reports';
import type { CashClosingsReportGroup } from '../../api/types';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { ClosingFigures } from './components/ClosingFigures/ClosingFigures';
import { ClosingsTable } from './components/ClosingsTable/ClosingsTable';

const groups: { value: CashClosingsReportGroup; label: string }[] = [
  { value: 'user', label: 'Cajero' },
  { value: 'location', label: 'Sede' },
  { value: 'day', label: 'Día' },
];
const groupParam = enumParam<CashClosingsReportGroup>(['user', 'location', 'day'], 'user');

/** /reportes/cierres: the period's cash closings by cashier, location or day, with shortages first. */
export function CashClosingsReportPage() {
  const params = useReportParams();
  const [group, setGroup] = useSearchParamsState('agrupar', groupParam);
  const query = { ...params.query, group_by: group };
  const report = useCashClosingsReport(query, !params.problem);

  return (
    <ReportSection
      title="Cierres de caja"
      description="Lo que vendió cada caja y si cuadró: los faltantes y sobrantes del arqueo."
      actions={<ExportButtons query={{ report: 'cash_closings', ...query }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params}>
        <Field label="Agrupar por">
          <Select
            options={groups}
            value={group}
            onValueChange={(value) => setGroup(groupParam.parse(value) ?? 'user')}
          />
        </Field>
      </ReportFilters>
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar los cierres"
          isEmpty={report.data?.totals.closings === 0}
          empty={
            <EmptyState
              icon={<IconCashStack />}
              title="Sin cierres en el periodo"
              description="Pruebe con otras fechas."
            />
          }
        >
          {() =>
            report.data && (
              <>
                <ClosingFigures totals={report.data.totals} />
                <ClosingsTable group={group} rows={report.data.items} />
              </>
            )
          }
        </QueryState>
      )}
    </ReportSection>
  );
}
