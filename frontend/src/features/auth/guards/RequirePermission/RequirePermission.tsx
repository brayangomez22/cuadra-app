import { Outlet } from 'react-router';
import { ButtonLink, EmptyState, IconAlert, Page } from '@/shared/ui';
import { useCan } from '../../session/useCan';
import { paths } from '@/app/routes/paths';

// RequirePermission guards the routes of a section the role cannot use, for
// someone who opens its URL directly: the menu already hides it.
export function RequirePermission({ permission }: { permission: string }) {
  const can = useCan();
  if (can(permission)) return <Outlet />;
  return (
    <Page title="Sin acceso">
      <EmptyState
        icon={<IconAlert />}
        title="No tiene permiso para ver esta sección"
        description="Si la necesita para su trabajo, pídale al dueño o al administrador que cambie su rol."
        action={
          <ButtonLink to={paths.home()} variant="secondary">
            Ir al inicio
          </ButtonLink>
        }
      />
    </Page>
  );
}
