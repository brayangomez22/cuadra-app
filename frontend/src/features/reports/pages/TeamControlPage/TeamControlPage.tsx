import { useCan } from '@/features/auth';
import { enumParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { EmptyState, Field, IconUsers, Select } from '@/shared/ui';
import { useTeamControlReport } from '../../api/reports';
import type { ReportBucket } from '../../api/types';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import type { TrendMetric } from '../../model/teamTrend';
import { TeamResult } from './components/TeamResult/TeamResult';

const buckets: { value: ReportBucket; label: string }[] = [
  { value: 'day', label: 'Día' },
  { value: 'week', label: 'Semana' },
  { value: 'month', label: 'Mes' },
];
const metrics: { value: TrendMetric; label: string }[] = [
  { value: 'voids', label: 'Anulaciones' },
  { value: 'discounts_amount', label: 'Monto descontado' },
  { value: 'cash_shortage', label: 'Faltantes de caja' },
  { value: 'adjustments', label: 'Ajustes de inventario' },
];
const bucketParam = enumParam<ReportBucket>(['week', 'day', 'month'], 'week');
const metricParam = enumParam<TrendMetric>(['voids', 'discounts_amount', 'cash_shortage', 'adjustments'], 'voids');

/** /reportes/equipo: voids, discounts, price changes, adjustments and cash differences by user, and their trend. */
export function TeamControlPage() {
  const params = useReportParams();
  const [bucket, setBucket] = useSearchParamsState('cada', bucketParam);
  const [metric, setMetric] = useSearchParamsState('cifra', metricParam);
  const query = { ...params.query, bucket };
  const report = useTeamControlReport(query, !params.problem);
  const readsAudit = useCan()('audit:read');

  return (
    <ReportSection
      title="Control del equipo"
      description="Lo que hizo cada persona y cómo cambia en el tiempo, para ver lo que se sale de lo normal."
      actions={<ExportButtons query={{ report: 'team_control', ...query }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params}>
        <Field label="Tendencia por">
          <Select options={buckets} value={bucket} onValueChange={(v) => setBucket(bucketParam.parse(v) ?? 'week')} />
        </Field>
        <Field label="Cifra de la tendencia">
          <Select options={metrics} value={metric} onValueChange={(v) => setMetric(metricParam.parse(v) ?? 'voids')} />
        </Field>
      </ReportFilters>
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar el control del equipo"
          isEmpty={report.data?.users.length === 0}
          empty={
            <EmptyState
              icon={<IconUsers />}
              title="Nada que revisar"
              description="Nadie anuló, descontó ni ajustó en el periodo."
            />
          }
        >
          {() =>
            report.data && (
              <TeamResult
                report={report.data}
                metric={metric}
                metricLabel={metrics.find((m) => m.value === metric)?.label ?? ''}
                audit={readsAudit ? { desde: params.from, hasta: params.to } : undefined}
              />
            )
          }
        </QueryState>
      )}
    </ReportSection>
  );
}
