import { useState } from 'react';
import { describeError } from '@/shared/api/errors';
import { QueryState } from '@/shared/query';
import { Alert, Checkbox, Spinner } from '@/shared/ui';
import { useModules, useSetModuleActive } from '../../api/modules';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { moduleNeeds } from '../../model/moduleNeeds';
import styles from './ModulesSettings.module.scss';

/** /configuracion/modulos: the capabilities the business turns on (variants, lots...). Owner only. */
export function ModulesSettings() {
  const modules = useModules();
  const [error, setError] = useState<string | null>(null);
  const change = useSetModuleActive();

  const toggle = (id: string, active: boolean) => {
    setError(null);
    change.mutate({ id, active }, { onError: (err) => setError(describeError(err).message) });
  };

  return (
    <SettingsSection
      title="Módulos"
      description="Capacidades que su negocio activa según lo que vende. Desactivar un módulo no borra sus datos: vuelven al activarlo."
    >
      {error && <Alert tone="danger">{error}</Alert>}
      <QueryState
        queries={[modules]}
        errorTitle="No pudimos cargar los módulos"
        loading={<Spinner label="Cargando módulos…" />}
      >
        {() => (
          <ul className={styles.list}>
            {(modules.data ?? []).map((module, _, all) => (
              <li key={module.id} className={styles.item}>
                <Checkbox
                  label={module.name}
                  hint={moduleNeeds(module, all)}
                  checked={module.active}
                  disabled={change.isPending}
                  onChange={(event) => toggle(module.id, event.target.checked)}
                />
              </li>
            ))}
          </ul>
        )}
      </QueryState>
    </SettingsSection>
  );
}
