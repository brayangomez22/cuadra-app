/** The days of a report in Colombia, both included, as the API writes them. */
export type Period = { from: string; to: string };

// The API's longest period: a year, leap or not.
const MAX_DAYS = 366;
const DAY_MS = 24 * 60 * 60 * 1000;

/** The default period: this month so far. */
export function defaultPeriod(today: string): Period {
  return { from: `${today.slice(0, 8)}01`, to: today };
}

// A day of the API at midnight UTC, where a day is always 24 hours.
const utc = (day: string) => Date.parse(`${day}T00:00:00Z`);

/** What is wrong with a period, as the API would reject it (366 days at most), or null. */
export function periodProblem(from: string, to: string): string | null {
  if (!from || !to) return 'Elija el primer y el último día.';
  const days = (utc(to) - utc(from)) / DAY_MS + 1;
  if (days < 1) return 'El último día debe ser igual o posterior al primero.';
  if (days > MAX_DAYS) return 'El periodo puede ser de un año como máximo.';
  return null;
}
