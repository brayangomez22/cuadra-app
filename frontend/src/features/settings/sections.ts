import type { Gated } from '@/features/auth';

export type Section = Gated & { to: string; label: string };

// The settings sections, in the menu's order. The business's own settings and
// its modules are the owner's (tenant:manage); the catalog's, of whoever
// edits it (catalog:write). The variant axes belong to the variants module.
export const sections: Section[] = [
  { to: '/configuracion/general', label: 'General', permission: 'tenant:manage' },
  { to: '/configuracion/unidades', label: 'Unidades', permission: 'catalog:write' },
  { to: '/configuracion/atributos', label: 'Atributos', permission: 'catalog:write' },
  { to: '/configuracion/ejes', label: 'Ejes de variante', permission: 'catalog:write', module: 'variants' },
  { to: '/configuracion/modulos', label: 'Módulos', permission: 'tenant:manage' },
];
