import { exceptionSchema } from './exceptionSchema';

describe('exceptionSchema', () => {
  it.each([
    ['el ID de una ferretería', '01920000-0000-7000-8000-0000000000aa', true],
    ['el ID de un almacén de calzado, con espacios alrededor', ' 01920000-0000-7000-8000-0000000000bb ', true],
    ['un nombre en vez del ID', 'Ferretería El Tornillo', false],
    ['nada', '', false],
  ])('%s', (_, tenantId, valid) => {
    expect(exceptionSchema.safeParse({ tenant_id: tenantId, value: 'on' }).success).toBe(valid);
  });
});
