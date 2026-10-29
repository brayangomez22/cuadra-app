import { creditAuthorizationSchema } from './creditAuthorizationSchema';

describe('creditAuthorizationSchema', () => {
  it('toma el correo sin espacios y la contraseña tal cual', () => {
    expect(creditAuthorizationSchema.parse({ email: ' admin@ferreteria.co ', password: ' clave ' })).toEqual({
      email: 'admin@ferreteria.co',
      password: ' clave ',
    });
  });

  it.each([
    [{ email: '', password: 'clave' }, 'Escriba el correo del administrador'],
    [{ email: 'admin@boutique.co', password: '' }, 'Escriba la contraseña'],
  ])('pide los dos datos: %o', (values, message) => {
    expect(creditAuthorizationSchema.safeParse(values).error?.issues[0]?.message).toBe(message);
  });
});
