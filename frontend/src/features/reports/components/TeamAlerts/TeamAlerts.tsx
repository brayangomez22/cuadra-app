import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { Card, Skeleton } from '@/shared/ui';
import { useTeamWeeks } from '../../api/dashboard';
import { useDashboardDays } from '../../hooks/useDashboardDays';
import { weekAlerts } from '../../model/dashboard/weekAlerts';
import { UserAlerts } from './UserAlerts';
import styles from './TeamAlerts.module.scss';

/**
 * What someone of the team did this week above their normal (voids, discounts,
 * cash shortages), by person: the trend of the team report over the last eight
 * weeks, which each figure opens, and what that person did in the audit log.
 */
export function TeamAlerts() {
  const days = useDashboardDays();
  const team = useTeamWeeks({ from: days.weeksFrom, to: days.today, bucket: 'week' });
  const readsAudit = useCan()('audit:read');
  const alerts = team.data ? weekAlerts(team.data, days.weekStart) : [];
  const users = [...new Map(alerts.map((alert) => [alert.userId, alert.name])).entries()];

  return (
    <Card title="Control del equipo esta semana" className={styles.root}>
      <QueryState
        queries={[team]}
        errorTitle="No pudimos revisar el control del equipo"
        loading={<Skeleton shape="block" className={styles.loading} />}
        isEmpty={alerts.length === 0}
        empty={<p className={styles.quiet}>Nada fuera de lo normal esta semana.</p>}
      >
        <p className={styles.quiet}>Lo que pasa del doble de lo habitual de cada persona en las últimas 8 semanas.</p>
        <ul className={styles.list}>
          {users.map(([userId, name]) => (
            <UserAlerts
              key={userId}
              name={name}
              alerts={alerts.filter((alert) => alert.userId === userId)}
              trend={(cifra) => paths.reports.team({ desde: days.weeksFrom, hasta: days.today, cada: 'week', cifra })}
              audit={
                readsAudit ? paths.audit.list({ usuario: userId, desde: days.weekStart, hasta: days.today }) : null
              }
            />
          ))}
        </ul>
      </QueryState>
    </Card>
  );
}
