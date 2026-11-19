import type { ComponentType } from 'react';
import type { Gated } from '@/features/auth';
import {
  IconBox,
  IconCart,
  IconReceipt,
  IconCash,
  IconChart,
  IconCashStack,
  IconClipboardCheck,
  IconFlag,
  IconHistory,
  IconHome,
  IconSettings,
  IconTag,
  IconTruck,
  IconUserKey,
  IconUsers,
  IconWallet,
} from '@/shared/ui';
import { paths } from '../routes/paths';

/**
 * A menu item. permission is the one the role needs (api/openapi.yaml,
 * x-permission); module, the activable module the tenant must have active.
 */
export type NavItem = Gated & {
  to: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  /** Pages of the item outside its URL: a drawer's report lives under /caja but belongs to the history. */
  owns?: RegExp;
};

// Menu of the app. An item hides when the role lacks its permission or the
// tenant its module; the API enforces both on every request behind the screen.
export const navItems: NavItem[] = [
  { to: paths.home(), label: 'Inicio', icon: IconHome },
  { to: paths.sales.pos(), label: 'Vender', icon: IconCart, permission: 'sales:create' },
  { to: paths.sales.cash(), label: 'Caja', icon: IconCash, permission: 'sales:create' },
  {
    to: paths.sales.cashHistory(),
    label: 'Cierres de caja',
    icon: IconCashStack,
    permission: 'cash:manage',
    owns: /^\/caja\/[0-9a-f-]{36}(\/cerrar)?$/,
  },
  { to: paths.sales.list(), label: 'Ventas', icon: IconReceipt, permission: 'sales:read' },
  { to: paths.products.list(), label: 'Productos', icon: IconTag, permission: 'catalog:read' },
  { to: paths.customers.list(), label: 'Clientes', icon: IconUsers, permission: 'customers:read' },
  { to: paths.receivables.aging(), label: 'Cartera', icon: IconWallet, permission: 'receivables:read' },
  { to: paths.purchases.list(), label: 'Compras', icon: IconTruck, permission: 'purchases:read' },
  { to: paths.inventory.stock(), label: 'Inventario', icon: IconBox, permission: 'inventory:read' },
  { to: paths.inventory.counts(), label: 'Toma física', icon: IconClipboardCheck, permission: 'inventory:count' },
  // Owners and admins read the business reports.
  { to: paths.reports.home(), label: 'Reportes', icon: IconChart, permission: 'reports:read' },
  { to: paths.users.list(), label: 'Usuarios', icon: IconUserKey, permission: 'users:manage' },
  // Only the owner reads who did what (audit:read).
  { to: paths.audit.list(), label: 'Auditoría', icon: IconHistory, permission: 'audit:read' },
  // Owners and admins configure the catalog; the owner also the business.
  { to: paths.settings.home(), label: 'Configuración', icon: IconSettings, permission: 'catalog:write' },
  // Only for the Cuadra team (PLATFORM_OPERATORS).
  { to: paths.platform.flags(), label: 'Feature flags', icon: IconFlag, operator: true },
];
