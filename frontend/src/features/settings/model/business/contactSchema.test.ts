import { contactSchema } from './contactSchema';

describe('contactSchema', () => {
  it.each([
    { business: 'ferretería', address: ' Calle 45 # 12-30, local 2 ', city: 'Medellín', phone: '604 444 1234' },
    { business: 'almacén de ropa', address: 'Cra. 7 # 20-15', city: 'Bogotá D.C.', phone: '+57 (601) 300-1234' },
    { business: 'sin datos todavía', address: '', city: '', phone: '' },
  ])('acepta los datos de contacto ($business), sin espacios a los lados', ({ address, city, phone }) => {
    expect(contactSchema.parse({ address, city, phone })).toEqual({
      address: address.trim(),
      city,
      phone,
    });
  });

  it.each([
    { field: 'address', value: 'a'.repeat(201), message: 'La dirección puede tener máximo 200 caracteres' },
    { field: 'city', value: 'a'.repeat(101), message: 'La ciudad puede tener máximo 100 caracteres' },
    {
      field: 'phone',
      value: 'tel. 3001234567',
      message: 'Escriba el teléfono solo con números, espacios, +, - o paréntesis',
    },
    { field: 'phone', value: '1'.repeat(31), message: 'El teléfono puede tener máximo 30 caracteres' },
  ])('rechaza $field inválido', ({ field, value, message }) => {
    const result = contactSchema.safeParse({ address: '', city: '', phone: '', [field]: value });
    expect(result.error?.issues.map((issue) => issue.message)).toEqual([message]);
  });
});
