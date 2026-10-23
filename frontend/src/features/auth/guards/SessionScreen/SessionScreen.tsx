import { Alert, Button, Spinner, Stack } from '@/shared/ui';
import styles from './SessionScreen.module.scss';

/** The whole page while the session is checked. */
export function SessionLoading() {
  return (
    <div className={styles.root}>
      <Spinner size="lg" label="Abriendo Cuadra…" />
    </div>
  );
}

/** The whole page when the session could not be checked (server error or no connection). */
export function SessionError({ retry }: { retry: () => void }) {
  return (
    <div className={styles.root}>
      <Stack gap="4" align="start" className={styles.error}>
        <Alert tone="danger" title="No pudimos abrir Cuadra">
          Revise el internet e intente de nuevo. Si sigue igual, puede ser un problema nuestro.
        </Alert>
        <Button onClick={retry}>Intentar de nuevo</Button>
      </Stack>
    </div>
  );
}
