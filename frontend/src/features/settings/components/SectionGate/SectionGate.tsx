import { Outlet } from 'react-router';
import { canSee, useAccess, type Gated } from '@/features/auth';
import { EmptyState, IconAlert } from '@/shared/ui';

/**
 * SectionGate guards a settings section someone opens by its URL: the menu
 * already hides it. It answers inside the settings layout, which already
 * has the page's title.
 */
export function SectionGate(gate: Gated) {
  const access = useAccess();
  if (canSee(gate, access)) return <Outlet />;
  if (gate.permission && !access.permissions.includes(gate.permission)) {
    return (
      <EmptyState
        icon={<IconAlert />}
        title="No tiene permiso para ver esta sección"
        description="Esta configuración es del dueño del negocio."
      />
    );
  }
  return (
    <EmptyState
      icon={<IconAlert />}
      title="Esta sección no está activa"
      description="Es de un módulo que su negocio no tiene activo. El dueño puede activarlo en Módulos."
    />
  );
}
