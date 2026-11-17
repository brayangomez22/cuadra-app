import { useId } from 'react';
import { Badge, Button, Checkbox } from '@/shared/ui';
import { useClearFlagException, useSetFlagGlobal } from '../../../../api/flags';
import type { FeatureFlag } from '../../../../api/types';
import { ExceptionForm } from '../ExceptionForm/ExceptionForm';
import styles from './FlagCard.module.scss';

/** A flag: its global value, the businesses with an exception and the form to add one. */
export function FlagCard({ flag }: { flag: FeatureFlag }) {
  const titleId = useId();
  const setGlobal = useSetFlagGlobal();
  const clear = useClearFlagException();
  return (
    <section className={styles.root} aria-labelledby={titleId}>
      <header>
        <h2 id={titleId} className={styles.key}>
          {flag.key}
        </h2>
        <p className={styles.description}>{flag.description}</p>
      </header>
      <Checkbox
        label="Prendida para todas las empresas"
        hint="Las empresas con una excepción usan su propio valor."
        checked={flag.enabled}
        disabled={setGlobal.isPending}
        onChange={(event) => setGlobal.mutate({ key: flag.key, enabled: event.target.checked })}
      />
      {flag.exceptions.length === 0 ? (
        <p className={styles.empty}>Ninguna empresa tiene una excepción.</p>
      ) : (
        <ul className={styles.exceptions}>
          {flag.exceptions.map((ex) => (
            <li key={ex.tenant_id} className={styles.exception}>
              <span className={styles.tenant}>{ex.tenant_name}</span>
              <Badge tone={ex.enabled ? 'success' : 'neutral'}>{ex.enabled ? 'Prendida' : 'Apagada'}</Badge>
              <Button
                variant="ghost"
                aria-label={`Quitar la excepción de ${ex.tenant_name}`}
                disabled={clear.isPending}
                onClick={() => clear.mutate({ key: flag.key, tenantId: ex.tenant_id })}
              >
                Quitar
              </Button>
            </li>
          ))}
        </ul>
      )}
      <ExceptionForm flagKey={flag.key} />
    </section>
  );
}
