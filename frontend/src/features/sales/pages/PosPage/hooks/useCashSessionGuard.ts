import { useMemo, useState } from 'react';
import { useAuth } from '@/features/auth';
import { useLocations, type Location } from '@/features/inventory';
import { useCurrentCashSession } from '../../../api/cashSessions';
import { loadLocation, saveLocation } from '../../../model/storage';

/** loading covers a failure too: QueryState shows it. */
export type GuardStatus = 'loading' | 'cash_required' | 'no_locations' | 'ready';

/**
 * Whether the POS can sell, and where. Sales go to the cashier's open cash
 * session, at its location; if the tenant requires one and there is none,
 * the screen asks to open it. Without a session, the device remembers the
 * location it sells from.
 */
export function useCashSessionGuard() {
  const auth = useAuth();
  const user = auth.status === 'authenticated' ? auth.user : undefined;
  const locationKey = `cuadra.pos.location.${user?.tenant_id ?? ''}`;
  const locations = useLocations();
  const cash = useCurrentCashSession();
  const [storedLocation, setStoredLocation] = useState(() => loadLocation(locationKey));

  const activeLocations = useMemo(() => (locations.data ?? []).filter((l) => l.active), [locations.data]);
  const session = cash.data?.session;
  // With an open drawer, the sale happens where the drawer is.
  const locationId = session ? session.location_id : chosenLocation(activeLocations, storedLocation);

  function chooseLocation(value: string) {
    setStoredLocation(value);
    saveLocation(locationKey, value);
  }

  return {
    status: guardStatus({ locations, cash }),
    queries: [locations, cash],
    locations: locations.data ?? [],
    activeLocations,
    session,
    locationId,
    chooseLocation,
  };
}

export type CashSessionGuard = ReturnType<typeof useCashSessionGuard>;

function chosenLocation(active: Location[], stored: string | undefined) {
  if (active.some((l) => l.id === stored)) return stored;
  const [only] = active;
  return active.length === 1 ? only?.id : undefined;
}

type Queries = {
  locations: ReturnType<typeof useLocations>;
  cash: ReturnType<typeof useCurrentCashSession>;
};

function guardStatus({ locations, cash }: Queries): GuardStatus {
  if (!locations.isSuccess || !cash.isSuccess) return 'loading';
  if (cash.data.required && !cash.data.session) return 'cash_required';
  if (!locations.data.some((l) => l.active)) return 'no_locations';
  return 'ready';
}
