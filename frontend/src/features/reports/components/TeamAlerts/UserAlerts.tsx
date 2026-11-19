import { formatMoney } from '@/shared/format';
import { TextLink } from '@/shared/ui';
import type { AlertMetric, WeekAlert } from '../../model/dashboard/weekAlerts';
import styles from './TeamAlerts.module.scss';

type Props = { name: string; alerts: WeekAlert[]; trend: (metric: AlertMetric) => string; audit: string | null };

// What each figure is called in a sentence and in its link.
const figures: Record<AlertMetric, { text: (value: string) => string; noun: string }> = {
  voids: { text: (n) => `${n} ${n === '1' ? 'anulación' : 'anulaciones'}`, noun: 'anulaciones' },
  discounts_amount: { text: (amount) => `${formatMoney(amount)} en descuentos`, noun: 'descuentos' },
  cash_shortage: { text: (amount) => `${formatMoney(amount)} de faltantes de caja`, noun: 'faltantes de caja' },
};

/** A person's alerts of the week: each figure opens its trend; the audit link, what they did. */
export function UserAlerts({ name, alerts, trend, audit }: Props) {
  return (
    <li className={styles.user}>
      <strong>{name}</strong>
      <span className={styles.figures}>
        {alerts.map(({ metric, value }) => (
          <TextLink
            key={metric}
            to={trend(metric)}
            aria-label={`Ver la tendencia de ${figures[metric].noun} de ${name}`}
          >
            {figures[metric].text(value)}
          </TextLink>
        ))}
      </span>
      {audit && (
        <TextLink to={audit} tone="quiet" aria-label={`Ver lo que hizo ${name} esta semana`}>
          Ver lo que hizo
        </TextLink>
      )}
    </li>
  );
}
