import type { SupplierValues } from './supplierSchema';
import { supplierSchema } from './supplierSchema';
import { toSupplierInput } from './toSupplierInput';

function values(overrides: Partial<SupplierValues> = {}): SupplierValues {
  return {
    name: ' Distribuidora Andina SAS ',
    nit: '900.123.456-8',
    contact_name: '',
    email: '',
    phone: '',
    payment_term_days: 30,
    active: true,
    ...overrides,
  };
}

describe('toSupplierInput', () => {
  it('deja fuera los datos de contacto vacíos', () => {
    expect(toSupplierInput(values())).toEqual({
      name: 'Distribuidora Andina SAS',
      nit: '900.123.456-8',
      payment_term_days: 30,
      active: true,
    });
  });

  it('envía el contacto de un proveedor de calzado con su plazo', () => {
    const input = toSupplierInput(
      values({
        name: 'Calzado del Valle',
        contact_name: ' Marta Ríos ',
        email: 'ventas@calzadovalle.co',
        phone: '3001234567',
      }),
    );
    expect(input).toMatchObject({ contact_name: 'Marta Ríos', email: 'ventas@calzadovalle.co', phone: '3001234567' });
  });
});

describe('supplierSchema', () => {
  const parse = (fields: Record<string, unknown>) =>
    supplierSchema.safeParse({ ...values(), payment_term_days: '30', ...fields });

  it.each([
    ['exige el nombre', { name: '  ' }, 'Escriba el nombre del proveedor'],
    ['exige el NIT', { nit: '' }, 'Escriba el NIT con su dígito de verificación'],
    ['rechaza un plazo de más de 365 días', { payment_term_days: '400' }, 'El plazo va de 0 a 365 días'],
    ['rechaza un plazo con decimales', { payment_term_days: '15,5' }, 'El plazo va de 0 a 365 días'],
    ['rechaza un correo no válido', { email: 'ventas@' }, 'Escriba un correo válido'],
  ])('%s', (_, fields, message) => {
    const result = parse(fields);
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe(message);
  });

  it('lee el plazo como número de días y vacío como 0', () => {
    expect(parse({ payment_term_days: ' 45 ' }).data?.payment_term_days).toBe(45);
    expect(parse({ payment_term_days: '' }).data?.payment_term_days).toBe(0);
  });
});
