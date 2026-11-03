import { useLocations } from '@/features/inventory';
import { useOpenCashSessions } from '../api/cashSessions';
import type { CashSession } from '../api/types';
import { drawerOwner } from '../model/cash/drawerOwner';

/**
 * Whose an open drawer is, where and since when, to say it before closing it.
 * The drawer only has its user's id; the name comes from the drawers open now.
 * Without it (a deleted user, the list failed, or the user cannot see other
 * drawers: enabled false) cashier is undefined.
 */
export function useDrawerOwner(session: CashSession | undefined, { enabled = true } = {}) {
  const open = useOpenCashSessions({ enabled }).data?.items ?? [];
  const locations = useLocations().data ?? [];
  return session && drawerOwner(session, { open, locations });
}
