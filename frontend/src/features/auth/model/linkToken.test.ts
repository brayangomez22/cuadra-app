import { tokenFromHash } from './linkToken';

const token = '01920000-0000-7000-8000-0000000000aa.c2VjcmV0by1kZS1wcnVlYmEtcGFyYS1lbC1lbmxhY2U';

describe('tokenFromHash', () => {
  it.each([
    ['el fragmento con el token', `#${token}`, token],
    ['el token sin #', token, token],
    ['un fragmento vacío', '', null],
    ['un enlace cortado al copiarlo', `#${token.slice(0, 50)}`, null],
    ['algo que no es un token', '#hola', null],
  ])('%s', (_, hash, want) => {
    expect(tokenFromHash(hash)).toBe(want);
  });
});
