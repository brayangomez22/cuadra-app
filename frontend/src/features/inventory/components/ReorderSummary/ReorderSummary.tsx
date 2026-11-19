import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Card, Skeleton, TextLink } from '@/shared/ui';
import { useReorderAlerts } from '../../api/reorder';
import styles from './ReorderSummary.module.scss';

/**
 * How many products reached their minimum, adding up every location, for the
 * home screen, with a link to the list of all of them. Only the total is
 * needed: one alert per page. Nothing for whoever does not see inventory.
 */
export function ReorderSummary() {
  const allowed = useCan()('inventory:read');
  const alerts = useReorderAlerts({ limit: 1, enabled: allowed, live: true });
  if (!allowed) return null;

  const total = alerts.data?.total ?? 0;
  const link = <TextLink to={paths.inventory.reorder({ sede: 'todas' })}>Ver qué pedir</TextLink>;
  return (
    <Card title="Por reponer" action={total > 0 ? link : undefined}>
      <QueryState
        queries={[alerts]}
        errorTitle="No pudimos revisar lo que hay por reponer"
        loading={<Skeleton shape="block" className={styles.loading} />}
        isEmpty={total === 0}
        empty={<p className={styles.quiet}>Nada por reponer.</p>}
      >
        <div className={styles.root}>
          <p className={styles.figure}>{`${String(total)} ${total === 1 ? 'producto' : 'productos'}`}</p>
          <p className={styles.quiet}>Llegaron a su stock mínimo, sumando todas las sedes.</p>
        </div>
      </QueryState>
    </Card>
  );
}
