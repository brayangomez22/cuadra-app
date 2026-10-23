import type { ReactNode } from 'react';
import { describeError } from '@/shared/api/errors';
import { Alert, Button, Spinner } from '@/shared/ui';
import styles from './QueryState.module.scss';

/** What QueryState reads from a TanStack query (a UseQueryResult fits). */
export type QueryStatus = { isPending: boolean; isError: boolean; error: unknown; refetch: () => unknown };

type Props = {
  queries: QueryStatus[];
  /** Says what could not be loaded: "No pudimos cargar las existencias". */
  errorTitle: string;
  /** Defaults to a Spinner that announces "Cargando…". */
  loading?: ReactNode;
  /** Whether the loaded data has nothing to show; then `empty` is shown. */
  isEmpty?: boolean;
  empty?: ReactNode;
  /** A function runs only once the data is there. */
  children: ReactNode | (() => ReactNode);
};

/**
 * The loading, error and empty states of one or more queries, then its
 * children. An error wins over loading: the user can act on it right away.
 */
export function QueryState({ queries, errorTitle, loading, isEmpty = false, empty, children }: Props) {
  const failed = queries.filter((query) => query.isError);
  const [firstFailed] = failed;
  if (firstFailed) {
    return (
      <div className={styles.root}>
        <Alert tone="danger" title={errorTitle}>
          {describeError(firstFailed.error).message}
        </Alert>
        <Button variant="secondary" onClick={() => failed.forEach((query) => void query.refetch())}>
          Reintentar
        </Button>
      </div>
    );
  }
  if (queries.some((query) => query.isPending)) return loading ?? <Spinner className={styles.loading} />;
  if (isEmpty) return empty;
  return typeof children === 'function' ? children() : children;
}
