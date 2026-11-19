import { toUnits } from '@/shared/money/decimal';
import type { TeamControlReport } from '../api/types';

/** The figures of the team's control that a trend can follow. */
export type TrendMetric = 'voids' | 'discounts_amount' | 'cash_shortage' | 'adjustments';

export type TrendCell = { bucket: string; value: string; unusual: boolean };
export type TrendRow = { userId: string; name: string; cells: TrendCell[] };

// A bucket stands out when it is more than this many times the user's average.
const UNUSUAL = 2n;

/**
 * Each user's metric in each bucket of the report (a bucket without data is
 * zero), with what stands out from that user's normal: more than twice their
 * average of the period. Exact arithmetic: amounts never become floats.
 */
export function trendTable(report: TeamControlReport, metric: TrendMetric): { buckets: string[]; rows: TrendRow[] } {
  const buckets = [...new Set(report.series.map((point) => point.bucket_start))].sort();
  const values = new Map(
    report.series.map((point) => [`${point.user_id}|${point.bucket_start}`, String(point[metric])]),
  );
  const rows = report.users.map((user) => {
    const series = buckets.map((bucket) => ({ bucket, value: values.get(`${user.user_id}|${bucket}`) ?? '0' }));
    const total = series.reduce((sum, cell) => sum + toUnits(cell.value), 0n);
    const count = BigInt(series.length);
    const cells = series.map((cell) => {
      const units = toUnits(cell.value);
      return { ...cell, unusual: units > 0n && units * count > UNUSUAL * total };
    });
    return { userId: user.user_id, name: user.name, cells };
  });
  return { buckets, rows };
}
