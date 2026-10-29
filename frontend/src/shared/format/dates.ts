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
