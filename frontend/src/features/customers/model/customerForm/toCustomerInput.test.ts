import { aBoutique, aBuilder, customerInput, segmentIds } from '@/test/fixtures';
import { customerDefaults } from './customerDefaults';
import { toCustomerInput } from './toCustomerInput';

describe('toCustomerInput', () => {
  it('omite los campos de contacto vacíos y el segmento sin elegir', () => {
    // Ferretería: un cliente nuevo con lo mínimo.
    const values = { ...customerDefaults(), document_number: ' 1020304050 ', name: ' Carlos Pérez ' };
    expect(toCustomerInput(values)).toEqual({
      person_type: 'natural',
      document_type: 'cc',
      document_number: '1020304050',
      name: 'Carlos Pérez',
      tax_responsibilities: ['R-99-PN'],
      active: true,
    });
  });

  it.each([
    { business: 'ferretería', customer: aBuilder() },
    { business: 'almacén de ropa', customer: aBoutique() },
  ])('un cliente de $business vuelve igual a la API después de editarlo', ({ customer }) => {
    expect(toCustomerInput(customerDefaults(customer))).toEqual(customerInput(customer));
  });

  it('envía el segmento elegido', () => {
    const values = { ...customerDefaults(), document_number: '1', name: 'Ana', segment_id: segmentIds.wholesale };
    expect(toCustomerInput(values).segment_id).toBe(segmentIds.wholesale);
  });
});
