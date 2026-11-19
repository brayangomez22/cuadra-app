import { Outlet } from 'react-router';
import { useAccess, visibleItems } from '@/features/auth';
import { SectionLayout } from '@/shared/ui';
import { sections } from '../../model/sections';

/** /configuracion: the sections the user may see, and the chosen one. */
export function SettingsLayout() {
  return (
    <SectionLayout
      title="Configuración"
      description="Cómo trabaja su negocio en Cuadra: unidades, campos de producto, segmentos de cliente y módulos."
      label="Secciones de configuración"
      sections={visibleItems(sections, useAccess())}
    >
      <Outlet />
    </SectionLayout>
  );
}
