import { roleOptions } from './roles';

describe('roleOptions', () => {
  it('lista los cuatro roles, del que más puede al que menos, con lo que hace cada uno', () => {
    expect(roleOptions.map((r) => [r.value, r.label])).toEqual([
      ['owner', 'Dueño'],
      ['admin', 'Administrador'],
      ['cashier', 'Cajero'],
      ['warehouse', 'Bodeguero'],
    ]);
    for (const role of roleOptions) expect(role.description).not.toBe('');
  });
});
