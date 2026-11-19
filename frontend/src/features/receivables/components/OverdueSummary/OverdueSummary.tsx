import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { formatMoney, isZero } from '@/shared/format';
import { QueryState } from '@/shared/query';
import { Card, Skeleton, TextLink } from '@/shared/ui';
import { useAgingTotals } from '../../api/aging';
import styles from './OverdueSummary.module.scss';

/**
 * What the customers owe past its due date, adding up all of them, and how
 * much of it is over 90 days, for the home screen: the totals of /cartera.
 * Nothing for whoever does not see the receivables.
 */
export function OverdueSummary() {
  const allowed = useCan()('receivables:read');
  const totals = useAgingTotals({ enabled: allowed });
  if (!allowed) return null;

  return (
    <Card title="Cartera vencida" action={<TextLink to={paths.receivables.aging()}>Ver la cartera</TextLink>}>
      <QueryState
        queries={[totals]}
        errorTitle="No pudimos cargar la cartera"
        loading={<Skeleton shape="block" className={styles.loading} />}
        isEmpty={totals.data ? isZero(totals.data.overdue) : false}
        empty={<p className={styles.quiet}>Nadie tiene cuentas vencidas.</p>}
      >
        {() =>
          totals.data && (
            <div className={styles.root}>
              <p className={styles.figure}>{formatMoney(totals.data.overdue)}</p>
              {!isZero(totals.data.over_90) && (
                <p className={styles.quiet}>{`${formatMoney(totals.data.over_90)} a más de 90 días`}</p>
              )}
            </div>
          )
        }
      </QueryState>
    </Card>
  );
}
