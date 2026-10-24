import { attributes } from '@/test/fixtures';
import { toAttributeCreate, toAttributeUpdate, toAttributeValues } from './attributeInput';

// Droguería: principio activo (texto); almacén de ropa: material (lista).
const text = {
  label: 'Principio activo',
  key: 'principio_activo',
  type: 'text',
  options: 'sobra',
  required: true,
} as const;
const list = {
  label: 'Material',
  key: 'material',
  type: 'list',
  options: 'Algodón\n PVC \n',
  required: false,
} as const;

describe('toAttributeValues', () => {
  it('para agregar, un atributo de texto vacío y opcional', () => {
    expect(toAttributeValues(null)).toEqual({ label: '', key: '', type: 'text', options: '', required: false });
  });

  it('para editar, sus valores con las opciones una por línea', () => {
    expect(toAttributeValues(attributes[1]!)).toEqual({
      label: 'Material',
      key: 'material',
      type: 'list',
      options: 'Algodón\nPVC',
      required: false,
    });
  });
});

describe('toAttributeCreate', () => {
  it('una lista lleva sus opciones, una por línea', () => {
    expect(toAttributeCreate(list)).toEqual({
      key: 'material',
      label: 'Material',
      type: 'list',
      options: ['Algodón', 'PVC'],
      required: false,
    });
  });

  it('los demás tipos no llevan opciones', () => {
    expect(toAttributeCreate(text)).toEqual({
      key: 'principio_activo',
      label: 'Principio activo',
      type: 'text',
      required: true,
    });
  });
});

describe('toAttributeUpdate', () => {
  it('envía la etiqueta, las opciones de una lista y si es obligatorio; nunca la clave ni el tipo', () => {
    expect(toAttributeUpdate(list)).toEqual({ label: 'Material', options: ['Algodón', 'PVC'], required: false });
    expect(toAttributeUpdate(text)).toEqual({ label: 'Principio activo', required: true });
  });
});
