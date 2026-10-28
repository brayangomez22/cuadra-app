import type { Gated } from '@/features/auth';
import { paths } from '@/app/routes/paths';

export type Section = Gated & { to: string; label: string };

// The settings sections, in the menu's order. The business's own settings and
// its modules are the owner's (tenant:manage); the catalog's, of whoever
// edits it (catalog:write); the customer segments, of whoever manages them
// (customers:manage). The variant axes belong to the variants module.
export const sections: Section[] = [
  { to: paths.settings.general(), label: 'General', permission: 'tenant:manage' },
  { to: paths.settings.units(), label: 'Unidades', permission: 'catalog:write' },
  { to: paths.settings.attributes(), label: 'Atributos', permission: 'catalog:write' },
  { to: paths.settings.variantAxes(), label: 'Ejes de variante', permission: 'catalog:write', module: 'variants' },
  { to: paths.settings.customerSegments(), label: 'Segmentos de cliente', permission: 'customers:manage' },
  { to: paths.settings.modules(), label: 'Módulos', permission: 'tenant:manage' },
];
