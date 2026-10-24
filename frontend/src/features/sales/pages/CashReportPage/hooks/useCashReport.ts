import type { ComponentProps } from 'react';
import { useAuth, type Me } from '@/features/auth';
import { useLocations } from '@/features/inventory';
import { useCashSession } from '../../../api/cashSessions';
import type { CashReport } from '../../../components/CashReport/CashReport';
import { loadPaper } from '../../../model/storage';

export type CashReportData = ComponentProps<typeof CashReport>;

/**
 * A cash session and, once it is closed, its report: the business, the
 * location, the cashier when known and the paper of this device.
 */
export function useCashReport(sessionId: string) {
  const session = useCashSession(sessionId);
  const locations = useLocations();
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;

  const data = session.data;
  const closing = data?.closing;
  if (!data || !closing) return { session, report: undefined };

  const report: CashReportData = {
    session: { ...data, closing },
    business: businessOf(user),
    location: locationName(locations.data, data.location_id),
    cashier: cashierName(user, data.user_id),
    paper: loadPaper(),
  };
  return { session, report };
}

function businessOf(user: Me | undefined) {
  return { name: user?.tenant_name ?? '', nit: user?.tenant_nit ?? '' };
}

function locationName(locations: { id: string; name: string }[] | undefined, id: string): string {
  return locations?.find((l) => l.id === id)?.name ?? '';
}

// Users have no endpoint yet: the cashier's name is known only for one's own drawer.
function cashierName(user: Me | undefined, cashierId: string): string | undefined {
  return user?.id === cashierId ? user.name : undefined;
}
