import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { Card, Skeleton, TextLink } from '@/shared/ui';
import { useSalesReport } from '../../api/reports';
import { useDashboardDays } from '../../hooks/useDashboardDays';
import { compareSales } from '../../model/dashboard/salesComparison';
import { SalesToday } from './SalesToday';
import styles from './TodaySales.module.scss';

/**
 * What the business sold today against the same day last week up to the same
 * time, with its number of sales and average ticket, for the home screen. The
 * same query as the sales report, which the link opens on today.
 */
export function TodaySales() {
  const days = useDashboardDays();
  const today = useSalesReport({ from: days.today, to: days.today, until: days.until });
  const lastWeek = useSalesReport({ from: days.lastWeek, to: days.lastWeek, until: days.until });
  const link = (
    <TextLink to={paths.reports.sales({ desde: days.today, hasta: days.today })}>Ver las ventas de hoy</TextLink>
  );

  return (
    <Card title="Ventas de hoy" action={link} className={styles.root}>
      <QueryState
        queries={[today, lastWeek]}
        errorTitle="No pudimos cargar las ventas de hoy"
        loading={<Skeleton shape="block" className={styles.loading} />}
      >
        {() =>
          today.data &&
          lastWeek.data && (
            <SalesToday sales={compareSales(today.data.totals, lastWeek.data.totals)} lastWeek={days.lastWeek} />
          )
        }
      </QueryState>
    </Card>
  );
}
