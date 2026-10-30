import { phoneLabel } from './phoneLabel';

describe('phoneLabel', () => {
  it.each([
    ['celular de un maestro de obra', '3330106699', '333 010 6699'],
    ['fijo de un almacén de ropa en Bogotá', '6012345678', '601 234 5678'],
    ['ya escrito con espacios', '310 555 1234', '310 555 1234'],
    ['con indicativo del país', '+57 3101234567', '+57 3101234567'],
    ['de otra longitud', '4441234', '4441234'],
  ])('%s: %s → %s', (_, phone, expected) => {
    expect(phoneLabel(phone)).toBe(expected);
  });
});
