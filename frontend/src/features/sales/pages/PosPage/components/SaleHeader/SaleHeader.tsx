import { paths } from '@/app/routes/paths';
import { Field, Select, TextLink } from '@/shared/ui';
import type { CashSessionGuard } from '../../hooks/useCashSessionGuard';
import styles from './SaleHeader.module.scss';

type Props = {
  guard: CashSessionGuard;
  /** Runs after a location is chosen: the focus goes back to the search. */
  onChosen: () => void;
};

/** Where the sale happens: the open drawer's location, or a choice when there are several. */
export function SaleHeader({ guard, onChosen }: Props) {
  const { session, locations, activeLocations, locationId } = guard;
  if (session) {
    return (
      <p className={styles.cash}>
        Caja abierta en {locations.find((l) => l.id === session.location_id)?.name}
        {' · '}
        <TextLink to={paths.sales.cash()}>Ver caja</TextLink>
      </p>
    );
  }
  if (activeLocations.length <= 1) return null;
  return (
    <Field label="Sede" className={styles.location}>
      <Select
        options={activeLocations.map((l) => ({ value: l.id, label: l.name }))}
        value={locationId}
        placeholder="Elija la sede"
        onValueChange={(value) => {
          guard.chooseLocation(value);
          onChosen();
        }}
      />
    </Field>
  );
}
