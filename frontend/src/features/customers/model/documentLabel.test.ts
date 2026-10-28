import { documentLabel } from './documentLabel';

describe('documentLabel', () => {
  it.each([
    // Ferretería: un maestro de obra con cédula.
    { document_type: 'cc', document_number: '1020304050', want: 'CC 1020304050' },
    // Almacén de ropa: una empresa mayorista.
    { document_type: 'nit', document_number: '901234567-7', want: 'NIT 901234567-7' },
    { document_type: 'ce', document_number: '456789', want: 'CE 456789' },
    { document_type: 'passport', document_number: 'AB123456', want: 'Pasaporte AB123456' },
  ] as const)('muestra el documento $want con su tipo', ({ want, ...customer }) => {
    expect(documentLabel(customer)).toBe(want);
  });
});
