import { documentLabel } from './documentLabel';

describe('documentLabel', () => {
  it.each([
    // Ferretería: un maestro de obra con cédula.
    { document_type: 'cc', document_number: '1020304050', want: 'CC 1020304050' },
    // Almacén de ropa: una empresa mayorista.
    { document_type: 'nit', document_number: '901234567-7', want: 'NIT 901234567-7' },
    { document_type: 'ce', document_number: '456789', want: 'CE 456789' },
    { document_type: 'passport', document_number: 'AB123456', want: 'Pasaporte AB123456' },
    // Papelería: un estudiante con tarjeta de identidad.
    { document_type: 'ti', document_number: '1098765432', want: 'TI 1098765432' },
    { document_type: 'ppt', document_number: '5432109', want: 'PPT 5432109' },
    // Almacén de ropa: un proveedor del exterior que también compra.
    { document_type: 'foreign_nit', document_number: 'B12345678', want: 'NIT extranjero B12345678' },
    { document_type: 'foreign_id', document_number: 'X1234567', want: 'Doc. extranjero X1234567' },
  ] as const)('muestra el documento $want con su tipo', ({ want, ...customer }) => {
    expect(documentLabel(customer)).toBe(want);
  });
});
