import { attributes } from '@/test/fixtures';
import { describeType, typeOptions } from './attributeTypes';

const [brand, material, weight, imported] = attributes;

describe('describeType', () => {
  it('una lista dice sus opciones (almacén de ropa: el material)', () => {
    expect(describeType(material!)).toBe('Lista: Algodón, PVC');
  });

  it('una lista de droguería dice sus opciones', () => {
    const form = { ...material!, key: 'forma', label: 'Forma farmacéutica', options: ['Tableta', 'Jarabe'] };
    expect(describeType(form)).toBe('Lista: Tableta, Jarabe');
  });

  it.each([
    { attribute: brand!, type: 'Texto' },
    { attribute: weight!, type: 'Número' },
    { attribute: imported!, type: 'Sí/no' },
  ])('los demás tipos dicen solo su nombre: $type', ({ attribute, type }) => {
    expect(describeType(attribute)).toBe(type);
  });
});

describe('typeOptions', () => {
  it('ofrece los cuatro tipos en orden', () => {
    expect(typeOptions.map((option) => option.label)).toEqual(['Texto', 'Número', 'Lista', 'Sí/no']);
  });
});
