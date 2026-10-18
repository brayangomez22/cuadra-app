import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { meQueryKey } from '@/features/auth';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import type { components } from '@/shared/api/schema';
import { Alert, Checkbox, Spinner } from '@/shared/ui';
import { SettingsSection } from './SettingsSection';
import styles from './ModulesSettings.module.scss';

type Module = components['schemas']['Module'];

const modulesKey = ['modules'] as const;

/** /configuracion/modulos: the capabilities the business turns on (variants, lots...). Owner only. */
export function ModulesSettings() {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const modules = useQuery({
    queryKey: modulesKey,
    queryFn: async () => unwrap(await api.GET('/api/v1/modules')).items,
  });
  const change = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) =>
      unwrap(await api.PUT('/api/v1/modules/{moduleId}', { params: { path: { moduleId: id } }, body: { active } })),
    onMutate: () => setError(null),
    onError: (err) => setError(describeError(err).message),
    // /me lists the active modules: the menus and fields follow it.
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: modulesKey }),
        queryClient.invalidateQueries({ queryKey: meQueryKey }),
      ]),
  });

  const names = new Map((modules.data ?? []).map((m) => [m.id, m.name]));
  const needs = (module: Module) =>
    module.depends_on.length > 0
      ? `Necesita: ${module.depends_on.map((id) => names.get(id) ?? id).join(', ')}`
      : undefined;

  return (
    <SettingsSection
      title="Módulos"
      description="Capacidades que su negocio activa según lo que vende. Desactivar un módulo no borra sus datos: vuelven al activarlo."
    >
      {error && <Alert tone="danger">{error}</Alert>}
      {modules.isPending && <Spinner label="Cargando módulos…" />}
      {modules.isError && <Alert tone="danger">{describeError(modules.error).message}</Alert>}
      {modules.data && (
        <ul className={styles.list}>
          {modules.data.map((module) => (
            <li key={module.id} className={styles.item}>
              <Checkbox
                label={module.name}
                hint={needs(module)}
                checked={module.active}
                disabled={change.isPending}
                onChange={(event) => change.mutate({ id: module.id, active: event.target.checked })}
              />
            </li>
          ))}
        </ul>
      )}
    </SettingsSection>
  );
}
