import { paths } from '@/app/routes/paths';
import { useAuth, useCan } from '@/features/auth';
import { ReorderSummary } from '@/features/inventory';
import { OverdueSummary } from '@/features/receivables';
import { TeamAlerts, TodaySales } from '@/features/reports';
import { OpenCashSessions } from '@/features/sales';
import { ButtonLink, EmptyState, IconCart, IconHome, Page } from '@/shared/ui';
import styles from './HomePage.module.scss';

/**
 * The home screen. Whoever reads the reports (the owner and the admin) gets
 * the dashboard: today's sales on top, then what to act on and the team's
 * week. Each card loads, fails and refreshes on its own. Everyone else gets a
 * greeting, what to reorder if they see inventory and the way to sell.
 */
export function HomePage() {
  const auth = useAuth();
  const can = useCan();
  const firstName = auth.status === 'authenticated' ? auth.user.name.split(' ')[0] : '';
  const title = firstName ? `Hola, ${firstName}` : 'Inicio';

  if (!can('reports:read')) {
    return (
      <Page title={title}>
        <ReorderSummary />
        <EmptyState
          icon={<IconHome />}
          title="Todo listo para trabajar"
          description="Use el menú para ir a cada sección."
          action={
            can('sales:create') && (
              <ButtonLink to={paths.sales.pos()} icon={<IconCart />}>
                Ir a vender
              </ButtonLink>
            )
          }
        />
      </Page>
    );
  }
  return (
    <Page title={title} description="Así va el negocio hoy. Las cifras se actualizan solas cada 5 minutos.">
      <div className={styles.root}>
        <TodaySales />
        <div className={styles.attention}>
          <OpenCashSessions />
          <ReorderSummary />
          <OverdueSummary />
        </div>
        <TeamAlerts />
      </div>
    </Page>
  );
}
