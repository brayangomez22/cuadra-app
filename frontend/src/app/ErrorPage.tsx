import { Button, ButtonLink, EmptyState, IconAlert, IconHome } from '@/shared/ui';
import styles from './ErrorPage.module.scss';
import { paths } from './routes/paths';

// CrashPage replaces the whole app after an unexpected error. It needs no router
// or providers: it is also the fallback when those fail.
export function CrashPage() {
  return (
    <main className={styles.root}>
      <EmptyState
        headingLevel={1}
        icon={<IconAlert />}
        title="Algo salió mal"
        description="Ocurrió un error inesperado en esta pantalla y ya nos llegó el aviso. Recargue la página para seguir trabajando."
        action={<Button onClick={() => window.location.reload()}>Recargar la página</Button>}
      />
    </main>
  );
}

// NotFoundPage answers a URL that matches no route.
export function NotFoundPage() {
  return (
    <main className={styles.root}>
      <EmptyState
        headingLevel={1}
        icon={<IconAlert />}
        title="No encontramos esta página"
        description="Puede que la dirección esté mal escrita o que la página ya no exista."
        action={
          <ButtonLink to={paths.home()} icon={<IconHome />}>
            Ir al inicio
          </ButtonLink>
        }
      />
    </main>
  );
}
