import { visibleItems } from './access';

// Items as a menu declares them: a core screen, one behind a permission and
// screens of activable modules.
const items = [
  { to: '/', label: 'Inicio' },
  { to: '/productos', label: 'Productos', permission: 'catalog:read' },
  { to: '/configuracion/ejes', label: 'Ejes de variante', permission: 'catalog:write', module: 'variants' },
  { to: '/lotes', label: 'Lotes', permission: 'inventory:read', module: 'lots_expiry' },
];

function labels(list: { label: string }[]) {
  return list.map((item) => item.label);
}

describe('visibleItems', () => {
  it('un menú de un módulo inactivo no aparece', () => {
    // Ferretería: sin variantes ni lotes.
    const access = { permissions: ['catalog:read', 'catalog:write', 'inventory:read'], modules: [] };

    expect(labels(visibleItems(items, access))).toEqual(['Inicio', 'Productos']);
  });

  it('el menú de un módulo aparece cuando la empresa lo tiene activo', () => {
    // Almacén de ropa con variantes; droguería con lotes.
    const permissions = ['catalog:read', 'catalog:write', 'inventory:read'];

    expect(labels(visibleItems(items, { permissions, modules: ['variants'] }))).toEqual([
      'Inicio',
      'Productos',
      'Ejes de variante',
    ]);
    expect(labels(visibleItems(items, { permissions, modules: ['lots_expiry'] }))).toEqual([
      'Inicio',
      'Productos',
      'Lotes',
    ]);
  });

  it('el menú de un módulo activo no aparece sin el permiso que pide', () => {
    const access = { permissions: ['catalog:read', 'inventory:read'], modules: ['variants', 'lots_expiry'] };

    expect(labels(visibleItems(items, access))).toEqual(['Inicio', 'Productos', 'Lotes']);
  });
});
