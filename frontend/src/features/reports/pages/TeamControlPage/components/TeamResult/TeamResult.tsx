import type { TeamControlReport } from '../../../../api/types';
import { trendTable, type TrendMetric } from '../../../../model/teamTrend';
import { TeamTable } from '../TeamTable/TeamTable';
import { TrendTable } from '../TrendTable/TrendTable';

type Props = {
  report: TeamControlReport;
  metric: TrendMetric;
  metricLabel: string;
  audit?: { desde: string; hasta: string };
};

/** Each user's totals to compare them, and the chosen figure's trend with what stands out. */
export function TeamResult({ report, metric, metricLabel, audit }: Props) {
  const trend = trendTable(report, metric);
  return (
    <>
      <TeamTable users={report.users} audit={audit} />
      <TrendTable buckets={trend.buckets} rows={trend.rows} metric={metric} caption={`${metricLabel} en el tiempo`} />
    </>
  );
}
