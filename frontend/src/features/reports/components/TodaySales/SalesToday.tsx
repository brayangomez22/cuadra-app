import { formatMoney, weekdayName } from '@/shared/format';
import type { SalesComparison } from '../../model/dashboard/salesComparison';
import styles from './TodaySales.module.scss';

type Props = { sales: SalesComparison; lastWeek: string };

const arrows = { up: '▲', down: '▼', same: '=' } as const;

/** How the day compares with lastWeek, said in words: the arrow and the color only repeat it. */
function changeText({ change, percent, previous, sales }: SalesComparison, weekday: string): string {
  const then = `el ${weekday} pasado a esta hora`;
  if (percent === null)
    return change === 'same' ? `Ni hoy ni ${then} hubo ventas.` : `${capital(then)} no había ventas.`;
  // Early in the day, "100 % less" would alarm: what matters is where last week was.
  if (sales === 0) return `Aún no hay ventas hoy. ${capital(then)} llevaba ${formatMoney(previous)}.`;
  if (change === 'same') return `Igual que ${then} (${formatMoney(previous)})`;
  return `${percent} % ${change === 'up' ? 'más' : 'menos'} que ${then} (${formatMoney(previous)})`;
}

const capital = (text: string) => text.charAt(0).toUpperCase() + text.slice(1);

/** The day's total, big, its change against last week and the sales behind it. */
export function SalesToday({ sales, lastWeek }: Props) {
  return (
    <div className={styles.body}>
      <p className={styles.total}>{formatMoney(sales.total)}</p>
      <p className={styles.change} data-change={sales.change}>
        <span aria-hidden="true">{arrows[sales.change]}</span> {changeText(sales, weekdayName(lastWeek))}
      </p>
      <dl className={styles.figures}>
        <div>
          <dt>Ventas</dt>
          <dd>{sales.sales}</dd>
        </div>
        {sales.averageTicket !== null && (
          <div>
            <dt>Ticket promedio</dt>
            <dd>{formatMoney(sales.averageTicket)}</dd>
          </div>
        )}
      </dl>
    </div>
  );
}
