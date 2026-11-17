import { Outlet } from 'react-router';
import { ButtonLink, EmptyState, IconAlert, Page } from '@/shared/ui';
import { useAuth } from '../../session/useAuth';
import { paths } from '@/app/routes/paths';

// RequireOperator guards the screens of the Cuadra team (the feature flags).
// For anyone else they do not exist, as the API answers 404.
export function RequireOperator() {
  const auth = useAuth();
  if (auth.status === 'authenticated' && auth.user.operator) return <Outlet />;
  return (
    <Page title="Página no encontrada">
      <EmptyState
        icon={<IconAlert />}
        title="Esta página no existe"
        description="Revise la dirección o vuelva al inicio."
        action={
          <ButtonLink to={paths.home()} variant="secondary">
            Ir al inicio
          </ButtonLink>
        }
      />
    </Page>
  );
}
