import { groupLabel } from './rowLabels';

describe('groupLabel', () => {
  it.each([
    {
      name: 'un día se lee como en Colombia',
      group: 'day',
      key: '2026-11-03',
      label: '2026-11-03',
      expected: '03/11/2026',
    },
    {
      name: 'un medio de pago por su nombre',
      group: 'payment_method',
      key: 'nequi',
      label: 'nequi',
      expected: 'Nequi',
    },
    {
      name: 'una categoría de ferretería',
      group: 'category',
      key: 'c1',
      label: 'Herramientas',
      expected: 'Herramientas',
    },
    { name: 'una categoría de ropa', group: 'category', key: 'c2', label: 'Camisetas', expected: 'Camisetas' },
    { name: 'lo vendido sin categoría', group: 'category', key: '', label: '', expected: 'Sin categoría' },
    { name: 'un usuario que ya no existe', group: 'user', key: 'u1', label: '', expected: 'Usuario borrado' },
    { name: 'una sede por su nombre', group: 'location', key: 'l1', label: 'Bodega norte', expected: 'Bodega norte' },
  ] as const)('$name', ({ group, key, label, expected }) => {
    expect(groupLabel(group, { key, label })).toBe(expected);
  });
});
