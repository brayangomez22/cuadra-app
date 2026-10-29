// The days between two days of the API, counted on UTC midnights so a change
// of month, year or daylight saving time does not shift them.
const dayMs = 24 * 60 * 60 * 1000;
const daysBetween = (from: string, to: string) => Math.round((Date.parse(to) - Date.parse(from)) / dayMs);

export type DueState = { state: 'overdue' | 'due_today' | 'upcoming'; days: number };

/** Whether a receivable due on dueDate is overdue on today (both days in Colombia), and by how many days. */
export function dueState(dueDate: string, today: string): DueState {
  const days = daysBetween(today, dueDate);
  if (days < 0) return { state: 'overdue', days: -days };
  return { state: days === 0 ? 'due_today' : 'upcoming', days };
}
