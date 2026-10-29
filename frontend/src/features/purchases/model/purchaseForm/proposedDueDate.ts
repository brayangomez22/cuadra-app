const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * The due date of a purchase on credit as the API proposes it: the
 * supplier's term in days after the invoice. Days are dates without a time
 * zone ("2026-10-20"), so the arithmetic is in UTC and never shifts a day.
 */
export function proposedDueDate(issuedOn: string, termDays: number): string {
  const issued = Date.parse(`${issuedOn}T00:00:00Z`);
  if (Number.isNaN(issued)) return '';
  return new Date(issued + termDays * DAY_MS).toISOString().slice(0, 10);
}
