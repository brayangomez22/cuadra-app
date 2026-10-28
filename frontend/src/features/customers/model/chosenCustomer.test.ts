import { aBoutique, aBuilder } from '@/test/fixtures';
import { chosenCustomer, quickCustomerDefaults } from './chosenCustomer';

describe('chosenCustomer', () => {
  it.each([
    { customer: aBuilder(), want: { name: 'Carlos Pérez', document: 'CC 1020304050' } },
    { customer: aBoutique(), want: { name: 'Confecciones Sol SAS', document: 'NIT 901234567-7' } },
  ])('guarda el nombre y el documento de $want.name', ({ customer, want }) => {
    expect(chosenCustomer(customer)).toEqual({ id: customer.id, ...want });
  });
});

describe('quickCustomerDefaults', () => {
  it.each([
    { searched: ' 79888777 ', document_number: '79888777', name: '' },
    { searched: '900.123.456-8', document_number: '900.123.456-8', name: '' },
    { searched: 'Lucía Rojas', document_number: '', name: 'Lucía Rojas' },
    { searched: '', document_number: '', name: '' },
  ])('lo buscado «$searched» llena el documento o el nombre', ({ searched, document_number, name }) => {
    expect(quickCustomerDefaults(searched)).toMatchObject({ document_number, name, person_type: 'natural' });
  });
});
