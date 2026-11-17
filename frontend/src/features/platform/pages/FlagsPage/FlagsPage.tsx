import { QueryState } from '@/shared/query';
import { Page, Spinner, Stack } from '@/shared/ui';
import { useFeatureFlags } from '../../api/flags';
import { FlagCard } from './components/FlagCard/FlagCard';

/**
 * /interno/flags: the feature flags, for the Cuadra team only. A flag is a
 * temporary switch (release to a pilot, turn off what fails), not a module.
 */
export function FlagsPage() {
  const flags = useFeatureFlags();
  return (
    <Page
      title="Feature flags"
      description="Cada flag tiene un valor para todas las empresas y excepciones por empresa. Los cambios llegan a las demás instancias en unos segundos."
    >
      <QueryState
        queries={[flags]}
        errorTitle="No pudimos cargar las flags"
        loading={<Spinner label="Cargando flags…" />}
      >
        {() => (
          <Stack>
            {(flags.data ?? []).map((flag) => (
              <FlagCard key={flag.key} flag={flag} />
            ))}
          </Stack>
        )}
      </QueryState>
    </Page>
  );
}
