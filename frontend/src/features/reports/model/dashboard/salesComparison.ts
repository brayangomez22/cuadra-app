import { divRound2, fromUnits, percentOf, toUnits } from '@/shared/money/decimal';
import type { SalesReport } from '../../api/types';

type Totals = SalesReport['totals'];

/** Today's sales against the same day last week, up to the same time. */
export type SalesComparison = {
  total: string;
  previous: string;
  sales: number;
  change: 'up' | 'down' | 'same';
  /** How much today differs from last week, as a whole percent; null when last week sold nothing. */
  percent: string | null;
  /** What a sale averages today; null without sales. */
  averageTicket: string | null;
};

/** Compares two days' totals. Exact arithmetic: amounts never become floats. */
export function compareSales(today: Totals, previous: Totals): SalesComparison {
  const now = toUnits(today.total);
  const before = toUnits(previous.total);
  const difference = now > before ? now - before : before - now;
  return {
    total: today.total,
    previous: previous.total,
    sales: today.sales_count,
    change: now > before ? 'up' : now < before ? 'down' : 'same',
    percent: before > 0n ? String(percentOf(difference, before)) : null,
    averageTicket: today.sales_count > 0 ? fromUnits(divRound2(now, BigInt(today.sales_count))) : null,
  };
}
