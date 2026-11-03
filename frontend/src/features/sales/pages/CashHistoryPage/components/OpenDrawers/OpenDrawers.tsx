import { paths } from '@/app/routes/paths';
import { formatDateTime } from '@/shared/format';
import { ButtonLink } from '@/shared/ui';
import type { CashSessionSummary } from '../../../../api/types';
import styles from './OpenDrawers.module.scss';

type Props = { sessions: CashSessionSummary[]; locationName: (id: string) => string; myId: string | undefined };

/**
 * The drawers open right now, above the history: who, where and since when,
 * with a real button to close each one. Closing a forgotten drawer from a phone
 * at night is why the owner comes here. The user's own drawer closes from /caja.
 */
export function OpenDrawers({ sessions, locationName, myId }: Props) {
  if (sessions.length === 0) return null;
  const title = sessions.length === 1 ? '1 caja abierta' : `${String(sessions.length)} cajas abiertas`;
  return (
    <section className={styles.root} aria-label="Cajas abiertas">
      <h2 className={styles.title}>{title}</h2>
      <ul className={styles.list}>
        {sessions.map((s) => (
          <OpenDrawer key={s.id} session={s} location={locationName(s.location_id)} mine={s.user_id === myId} />
        ))}
      </ul>
    </section>
  );
}

function OpenDrawer({ session: s, location, mine }: { session: CashSessionSummary; location: string; mine: boolean }) {
  const cashier = s.cashier_name ?? 'Usuario eliminado';
  return (
    <li className={styles.item}>
      <div className={styles.who}>
        <strong>{mine ? `${cashier} (usted)` : cashier}</strong>
        <span className={styles.where}>
          {[location, `desde ${formatDateTime(s.opened_at)}`].filter(Boolean).join(' · ')}
        </span>
      </div>
      {mine ? (
        <ButtonLink to={paths.sales.closeCash()} variant="secondary">
          Cerrar mi caja
        </ButtonLink>
      ) : (
        <ButtonLink to={paths.sales.closeCashOf(s.id)} variant="secondary" aria-label={`Cerrar la caja de ${cashier}`}>
          Cerrar caja
        </ButtonLink>
      )}
    </li>
  );
}
