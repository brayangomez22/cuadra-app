import { joinLines, parseLines } from './lines';

describe('parseLines', () => {
  it.each([
    { business: 'almacén de ropa', text: 'S\nM\nL', lines: ['S', 'M', 'L'] },
    { business: 'droguería', text: '  Caja x 10 \n\nFrasco 120 ml\n  \n', lines: ['Caja x 10', 'Frasco 120 ml'] },
    { business: 'cualquiera', text: '\n \n', lines: [] },
  ])('toma las líneas con texto, sin espacios a los lados ($business)', ({ text, lines }) => {
    expect(parseLines(text)).toEqual(lines);
  });
});

describe('joinLines', () => {
  it('pone un valor por línea, para el cuadro de texto', () => {
    expect(joinLines(['Algodón', 'PVC'])).toBe('Algodón\nPVC');
    expect(joinLines([])).toBe('');
  });
});
