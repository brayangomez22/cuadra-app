import type { TeamControlReport } from '../../api/types';
import { trendTable, type TrendMetric } from '../teamTrend';

/** What the dashboard watches of the team: voids, discounts and cash shortages. */
export const alertMetrics = ['voids', 'discounts_amount', 'cash_shortage'] as const satisfies TrendMetric[];
export type AlertMetric = (typeof alertMetrics)[number];

/** Something a user did this week that is above their normal. */
export type WeekAlert = { userId: string; name: string; metric: AlertMetric; value: string };

/**
 * The alerts of the week that starts on weekStart: each figure of a user that
 * the team report's trend marks as out of their normal (more than twice their
 * average of the period, teamTrend), so the dashboard and the report agree.
 */
export function weekAlerts(report: TeamControlReport, weekStart: string): WeekAlert[] {
  const tables = alertMetrics.map((metric) => ({ metric, rows: trendTable(report, metric).rows }));
  return report.users.flatMap((user) =>
    tables.flatMap(({ metric, rows }) => {
      const cell = rows.find((row) => row.userId === user.user_id)?.cells.find((c) => c.bucket === weekStart);
      return cell?.unusual ? [{ userId: user.user_id, name: user.name, metric, value: cell.value }] : [];
    }),
  );
}
