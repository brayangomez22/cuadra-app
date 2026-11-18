import { linkUrl } from './linkUrl';

describe('linkUrl', () => {
  it('pone el token en el fragmento, que no viaja al servidor', () => {
    expect(linkUrl('/invitacion', 'tenant.secreto', 'https://app.cuadra.co')).toBe(
      'https://app.cuadra.co/invitacion#tenant.secreto',
    );
  });
});
