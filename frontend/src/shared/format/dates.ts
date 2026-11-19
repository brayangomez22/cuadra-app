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

/** A day of the API moved by days (negative goes back): ("2026-03-01", -1) → "2026-02-28". Computed in UTC, where a day is always 24 hours. */
export function addDays(day: string, days: number): string {
  const date = new Date(`${day}T00:00:00Z`);
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

/** The day before a day of the API: "2026-03-01" → "2026-02-28". */
export function previousDay(day: string): string {
  return addDays(day, -1);
}

/** The day of the week of a day of the API, Monday first: 0 is Monday and 6 is Sunday. */
export function weekdayOf(day: string): number {
  return (new Date(`${day}T00:00:00Z`).getUTCDay() + 6) % 7;
}

const bogotaClock = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'America/Bogota',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
});

/** The time it is in Colombia at now, as the API takes it: "09:05". */
export function clockInBogota(now = new Date()): string {
  return bogotaClock.format(now);
}

const weekdayFormat = new Intl.DateTimeFormat('es-CO', { weekday: 'long', timeZone: 'UTC' });

/** The name of the day of the week of a day of the API: "2026-11-19" → "jueves". */
export function weekdayName(day: string): string {
  return weekdayFormat.format(new Date(`${day}T00:00:00Z`));
}
