import type { Me } from '../api/types';

type Role = Me['role'];

/** The name of each role, as the user sees it. */
export const roleLabels: Record<Role, string> = {
  owner: 'Dueño',
  admin: 'Administrador',
  cashier: 'Cajero',
  warehouse: 'Bodeguero',
};

const roleDescriptions: Record<Role, string> = {
  owner: 'Puede hacer todo, incluida la configuración de la empresa.',
  admin: 'Maneja el catálogo, el inventario, las compras, los clientes y la cartera.',
  cashier: 'Vende, maneja su caja y recibe abonos de los clientes.',
  warehouse: 'Registra entradas, ajusta y cuenta el inventario.',
};

/** The roles, from the one who can do most to the one who can do least, with what each does. */
export const roleOptions = (Object.keys(roleLabels) as Role[]).map((role) => ({
  value: role,
  label: roleLabels[role],
  description: roleDescriptions[role],
}));
