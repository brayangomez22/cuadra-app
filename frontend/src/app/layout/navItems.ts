import type { ComponentType } from 'react';
import { IconBox, IconCart, IconCash, IconHome, IconTag } from '@/shared/ui';

export type NavItem = {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  /** Permission the role needs to see the item (api/openapi.yaml, x-permission). */
  permission?: string;
};

// Menu of the app. An item hides when the role lacks its permission; the API
// enforces the same permission on every request behind the screen.
export const navItems: NavItem[] = [
  { to: '/', label: 'Inicio', icon: IconHome },
  { to: '/vender', label: 'Vender', icon: IconCart, permission: 'sales:create' },
  { to: '/caja', label: 'Caja', icon: IconCash, permission: 'sales:create' },
  { to: '/productos', label: 'Productos', icon: IconTag, permission: 'catalog:read' },
  { to: '/inventario', label: 'Inventario', icon: IconBox, permission: 'inventory:read' },
];
