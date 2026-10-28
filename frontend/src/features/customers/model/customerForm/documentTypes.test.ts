import { documentTypeOptions } from './documentTypes';

describe('documentTypeOptions', () => {
  it('una persona jurídica solo ofrece NIT', () => {
    expect(documentTypeOptions('legal').map((o) => o.value)).toEqual(['nit']);
  });

  it('una persona natural ofrece CC, NIT, CE y pasaporte', () => {
    expect(documentTypeOptions('natural').map((o) => o.value)).toEqual(['cc', 'nit', 'ce', 'passport']);
  });
});
