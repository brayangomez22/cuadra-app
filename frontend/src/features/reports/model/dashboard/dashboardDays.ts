import { addDays, clockInBogota, todayInBogota, weekdayOf } from '@/shared/format';

/** The days the owner's dashboard reads, in Colombia, as the API writes them. */
export type DashboardDays = {
  today: string;
  /** The same day of the week, a week ago: today is compared with it up to the same time. */
  lastWeek: string;
  /** The time it is now (HH:MM): where both days are cut. */
  until: string;
  /** The Monday of this week: its alerts are the ones on the dashboard. */
  weekStart: string;
  /** Seven weeks before weekStart: with this week, the eight the team's normal is measured on. */
  weeksFrom: string;
};

// The weeks a user's normal is the average of, counting this one.
const WEEKS = 8;

/** The days of the dashboard at now. */
export function dashboardDays(now: Date): DashboardDays {
  const today = todayInBogota(now);
  const weekStart = addDays(today, -weekdayOf(today));
  return {
    today,
    lastWeek: addDays(today, -7),
    until: clockInBogota(now),
    weekStart,
    weeksFrom: addDays(weekStart, -7 * (WEEKS - 1)),
  };
}
