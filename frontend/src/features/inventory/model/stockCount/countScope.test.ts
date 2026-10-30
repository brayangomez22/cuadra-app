import { countScope } from './countScope';

const categories = [
  { id: 'a', name: 'Plomería' },
  { id: 'b', name: 'Camisetas' },
  { id: 'c', name: 'Calzado' },
];

describe('countScope', () => {
  it.each([
    ['sin categorías es toda la sede', [], 'Toda la sede'],
    ['nombra la categoría de un negocio', ['a'], 'Plomería'],
    ['nombra varias, en el orden del catálogo', ['b', 'a'], 'Plomería, Camisetas'],
    ['nombra las categorías de otro tipo de negocio', ['b', 'c'], 'Camisetas, Calzado'],
  ])('%s', (_, ids, expected) => {
    expect(countScope(ids, categories)).toBe(expected);
  });

  it('sin los nombres cargados dice que son algunas categorías', () => {
    expect(countScope(['a'], undefined)).toBe('Algunas categorías');
    expect(countScope(['z'], categories)).toBe('Algunas categorías');
  });
});
