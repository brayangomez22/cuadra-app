import type { DocumentType, PersonType } from '../../api/types';

export const personTypeOptions: { value: PersonType; label: string }[] = [
  { value: 'natural', label: 'Persona natural' },
  { value: 'legal', label: 'Persona jurídica' },
];

const documentTypeLabels: Record<DocumentType, string> = {
  cc: 'Cédula de ciudadanía',
  nit: 'NIT',
  ce: 'Cédula de extranjería',
  passport: 'Pasaporte',
  ti: 'Tarjeta de identidad',
  ppt: 'Permiso por protección temporal (PPT)',
  foreign_id: 'Documento de identificación extranjero',
  foreign_nit: 'NIT de otro país',
};

// The DIAN's identification documents, the most common first.
const allTypes: DocumentType[] = ['cc', 'nit', 'ce', 'passport', 'ti', 'ppt', 'foreign_id', 'foreign_nit'];
const companyTypes: DocumentType[] = ['nit', 'foreign_nit'];

/** The documents a person may have: a legal person is identified by a NIT, of Colombia or another country (the API checks it too). */
export function documentTypeOptions(person: PersonType) {
  const types = person === 'legal' ? companyTypes : allTypes;
  return types.map((value) => ({ value, label: documentTypeLabels[value] }));
}

/** Whether a legal person can be identified with the document. */
export function identifiesCompanies(type: DocumentType) {
  return companyTypes.includes(type);
}
