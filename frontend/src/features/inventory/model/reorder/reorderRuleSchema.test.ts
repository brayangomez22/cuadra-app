import { reorderRuleSchema } from './reorderRuleSchema';

function errors(min: string, max: string) {
  const result = reorderRuleSchema.safeParse({ min_quantity: min, max_quantity: max });
  return result.success ? {} : Object.fromEntries(result.error.issues.map((i) => [i.path.join('.'), i.message]));
}

describe('reorderRuleSchema', () => {
  it.each([
    { name: 'unidades de una tienda', min: '10', max: '50', expected: ['10', '50'] },
    { name: 'pares de calzado', min: '2', max: '6', expected: ['2', '6'] },
    { name: 'metros con coma decimal', min: '2,5', max: '30,75', expected: ['2.5', '30.75'] },
    { name: 'mínimo en cero e igual al máximo', min: '0', max: '0', expected: ['0', '0'] },
  ])('acepta $name y entrega el decimal de la API', ({ min, max, expected }) => {
    expect(reorderRuleSchema.parse({ min_quantity: min, max_quantity: max })).toEqual({
      min_quantity: expected[0],
      max_quantity: expected[1],
    });
  });

  it('pide los dos valores', () => {
    expect(errors('', '')).toEqual({
      min_quantity: 'Escriba el stock mínimo',
      max_quantity: 'Escriba el stock máximo',
    });
  });

  it('rechaza cantidades negativas', () => {
    expect(errors('-1', '5')).toEqual({ min_quantity: 'El stock mínimo no puede ser negativo' });
  });

  it('rechaza un máximo menor que el mínimo, con decimales', () => {
    expect(errors('8', '5')).toEqual({ max_quantity: 'El máximo no puede ser menor que el mínimo' });
    expect(errors('2,5', '2,4999')).toEqual({ max_quantity: 'El máximo no puede ser menor que el mínimo' });
  });
});
