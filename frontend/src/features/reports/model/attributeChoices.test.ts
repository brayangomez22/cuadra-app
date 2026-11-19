import { attributeChoices } from './attributeChoices';

const base = { id: 'a', required: false, options: [] as string[] };

describe('attributeChoices', () => {
  it('un atributo de lista (género, en ropa) ofrece sus opciones', () => {
    const genre = { ...base, key: 'genero', label: 'Género', type: 'list' as const, options: ['Mujer', 'Hombre'] };
    expect(attributeChoices(genre)).toEqual([
      { value: 'Mujer', label: 'Mujer' },
      { value: 'Hombre', label: 'Hombre' },
    ]);
  });

  it('uno de sí o no (venta con fórmula, en droguería) ofrece Sí y No con los valores que guarda la API', () => {
    const prescription = { ...base, key: 'con_formula', label: 'Con fórmula', type: 'boolean' as const };
    expect(attributeChoices(prescription)).toEqual([
      { value: 'true', label: 'Sí' },
      { value: 'false', label: 'No' },
    ]);
  });

  it('uno de texto o número (marca, en ferretería) se escribe', () => {
    expect(attributeChoices({ ...base, key: 'marca', label: 'Marca', type: 'text' })).toBeNull();
  });
});
