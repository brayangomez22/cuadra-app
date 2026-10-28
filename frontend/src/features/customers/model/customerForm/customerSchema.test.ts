import { customerDefaults } from './customerDefaults';
import { customerSchema } from './customerSchema';

const valid = { ...customerDefaults(), document_number: '1020304050', name: 'Carlos Pérez' };

// The message of the first issue on a field, or undefined if the values pass.
function issueOn(values: object, field: string) {
  const result = customerSchema.safeParse(values);
  return result.error?.issues.find((issue) => issue.path[0] === field)?.message;
}

describe('customerSchema', () => {
  it('acepta un cliente con documento y nombre', () => {
    expect(customerSchema.safeParse(valid).success).toBe(true);
  });

  it.each([
    {
      field: 'document_number',
      values: { ...valid, document_number: '  ' },
      message: 'Escriba el número de documento',
    },
    { field: 'name', values: { ...valid, name: '' }, message: 'Escriba el nombre' },
    { field: 'email', values: { ...valid, email: 'compras@' }, message: 'Escriba un correo válido' },
  ])('exige un $field válido', ({ field, values, message }) => {
    expect(issueOn(values, field)).toBe(message);
  });

  it('rechaza una persona jurídica que no se identifica con NIT', () => {
    const values = { ...valid, person_type: 'legal', document_type: 'cc', name: 'Confecciones Sol SAS' };
    expect(issueOn(values, 'document_type')).toBe('Una persona jurídica se identifica con NIT');
  });

  it('rechaza R-99-PN junto a otra responsabilidad', () => {
    const values = { ...valid, tax_responsibilities: ['R-99-PN', 'O-13'] };
    expect(issueOn(values, 'tax_responsibilities')).toBe('«No responsable» no se combina con las demás');
  });
});
