import { formatDateTime } from '@/shared/format';
import type { CashSession, CashSessionSummary } from '../../api/types';

/** Whose an open drawer is, where and since when; cashier is undefined when the name is not known. */
export type DrawerOwner = { cashier?: string; location: string; openedAt: string };

type Sources = { open: CashSessionSummary[]; locations: { id: string; name: string }[] };

/** The owner of a drawer from the drawers open now (which carry the name) and the locations. */
export function drawerOwner(session: CashSession, { open, locations }: Sources): DrawerOwner {
  return {
    cashier: open.find((s) => s.id === session.id)?.cashier_name,
    location: locations.find((l) => l.id === session.location_id)?.name ?? '',
    openedAt: formatDateTime(session.opened_at),
  };
}

// Ends a sentence once: a time already ends in "a. m.".
const sentence = (text: string) => (text.endsWith('.') ? `${text} ` : `${text}. `);

/** "Sede norte · abierta el 13/10/2026, 8:00 a. m.", or "" without an owner. */
export function whereAndSince(owner: DrawerOwner | undefined): string {
  if (!owner) return '';
  return [owner.location, `abierta el ${owner.openedAt}`].filter(Boolean).join(' · ');
}

/** "La abrió Pedro Ruiz en Sede norte el 13/10/2026, 8:00 a. m. ", or "" when the name is not known. */
export function openedBy(owner: DrawerOwner | undefined): string {
  if (!owner?.cashier) return '';
  const where = owner.location ? ` en ${owner.location}` : '';
  return sentence(`La abrió ${owner.cashier}${where} el ${owner.openedAt}`);
}

/** whereAndSince as the start of a paragraph, or "". */
export function whereAndSinceSentence(owner: DrawerOwner | undefined): string {
  const text = whereAndSince(owner);
  return text && sentence(text);
}
