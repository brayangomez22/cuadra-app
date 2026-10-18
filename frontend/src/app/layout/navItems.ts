import type { ComponentType } from 'react';
import type { Gated } from '@/features/auth';
import { IconBox, IconCart, IconCash, IconHome, IconSettings, IconTag } from '@/shared/ui';

/**
 * A menu item. permission is the one the role needs (api/openapi.yaml,
 * x-permission); module, the activable module the tenant must have active.
 */
export type NavItem = Gated & {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
};

// Menu of the app. An item hides when the role lacks its permission or the
// tenant its module; the API enforces both on every request behind the screen.
export const navItems: NavItem[] = [
  { to: '/', label: 'Inicio', icon: IconHome },
  { to: '/vender', label: 'Vender', icon: IconCart, permission: 'sales:create' },
  { to: '/caja', label: 'Caja', icon: IconCash, permission: 'sales:create' },
  { to: '/productos', label: 'Productos', icon: IconTag, permission: 'catalog:read' },
  { to: '/inventario', label: 'Inventario', icon: IconBox, permission: 'inventory:read' },
  // Owners and admins configure the catalog; the owner also the business.
  { to: '/configuracion', label: 'Configuración', icon: IconSettings, permission: 'catalog:write' },
];
