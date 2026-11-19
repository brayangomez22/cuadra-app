import { useAuth, useCan } from '@/features/auth';
import { useLocations } from '@/features/inventory';
import { QueryState } from '@/shared/query';
import { Card, Skeleton } from '@/shared/ui';
import { useOpenCashSessionsLive } from '../../api/cashSessions';
import { OpenDrawerList } from '../OpenDrawers/OpenDrawers';
import styles from './OpenCashSessions.module.scss';

/**
 * The drawers open right now, with who, where and since when, and the button
 * to close each one, for the home screen: the forgotten drawer is found in
 * the morning, not at the next count. Nothing for whoever does not manage cash.
 */
export function OpenCashSessions() {
  const allowed = useCan()('cash:manage');
  const sessions = useOpenCashSessionsLive({ enabled: allowed });
  const locations = useLocations().data ?? [];
  const auth = useAuth();
  if (!allowed) return null;

  const myId = auth.status === 'authenticated' ? auth.user.id : undefined;
  const locationName = (id: string) => locations.find((l) => l.id === id)?.name ?? '';
  return (
    <Card title="Cajas abiertas">
      <QueryState
        queries={[sessions]}
        errorTitle="No pudimos cargar las cajas abiertas"
        loading={<Skeleton shape="block" className={styles.loading} />}
        isEmpty={sessions.data?.length === 0}
        empty={<p className={styles.quiet}>No hay cajas abiertas.</p>}
      >
        {() => sessions.data && <OpenDrawerList sessions={sessions.data} locationName={locationName} myId={myId} />}
      </QueryState>
    </Card>
  );
}
