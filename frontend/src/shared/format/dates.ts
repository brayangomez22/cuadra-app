// Days of the API ("2026-11-05"): a due date is a day in Colombia, not an
// instant, so it is never shifted by a time zone.

const bogotaDay = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'America/Bogota',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
});

/** A day of the API as Colombians read it: "2026-11-05" → "05/11/2026". */
export function formatDay(day: string): string {
  const [year, month, date] = day.split('-');
  return `${date ?? ''}/${month ?? ''}/${year ?? ''}`;
}

/** The day it is in Colombia at now, as the API writes days: "2026-10-28". */
export function todayInBogota(now = new Date()): string {
  return bogotaDay.format(now);
}

/** The day before a day of the API: "2026-03-01" → "2026-02-28". Computed in UTC, where a day is always 24 hours. */
export function previousDay(day: string): string {
  const date = new Date(`${day}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() - 1);
  return date.toISOString().slice(0, 10);
}
