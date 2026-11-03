import type { ReactNode } from 'react';
import { paths } from '@/app/routes/paths';
import { QueryState } from '@/shared/query';
import { ButtonLink, EmptyState, IconBox, Spinner } from '@/shared/ui';
import type { useStockFormSetup } from '../../hooks/useStockFormSetup';
import styles from './StockFormState.module.scss';

type Props = {
  setup: ReturnType<typeof useStockFormSetup>;
  /** How many active locations the form needs. */
  minLocations?: number;
  emptyTitle: string;
  emptyDescription: string;
  children: () => ReactNode;
};

/** Loads what a form that moves stock starts with, or says why it cannot be used yet. */
export function StockFormState({ setup, minLocations = 1, emptyTitle, emptyDescription, children }: Props) {
  const loading = <Spinner label="Cargando…" className={styles.loading} />;
  return (
    <QueryState
      queries={[setup.locations]}
      errorTitle="No pudimos cargar las sedes"
      loading={loading}
      isEmpty={setup.options.length < minLocations}
      empty={
        <EmptyState
          icon={<IconBox />}
          title={emptyTitle}
          description={emptyDescription}
          action={
            <ButtonLink to={paths.inventory.stock()} variant="secondary">
              Ir a inventario
            </ButtonLink>
          }
        />
      }
    >
      {() => (setup.presetPending ? loading : children())}
    </QueryState>
  );
}
