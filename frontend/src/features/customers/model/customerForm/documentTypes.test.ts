import { documentTypeOptions } from './documentTypes';

describe('documentTypeOptions', () => {
  it('una persona jurídica ofrece NIT y NIT de otro país', () => {
    expect(documentTypeOptions('legal').map((o) => o.value)).toEqual(['nit', 'foreign_nit']);
  });

  it('una persona natural ofrece todos los documentos de la DIAN', () => {
    expect(documentTypeOptions('natural').map((o) => o.value)).toEqual([
      'cc',
      'nit',
      'ce',
      'passport',
      'ti',
      'ppt',
      'foreign_id',
      'foreign_nit',
    ]);
  });
});
