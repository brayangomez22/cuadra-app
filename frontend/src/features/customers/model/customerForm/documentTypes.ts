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
};

const allTypes: DocumentType[] = ['cc', 'nit', 'ce', 'passport'];

/** The documents a person may have: a legal person is identified by its NIT (the API checks it too). */
export function documentTypeOptions(person: PersonType) {
  const types = person === 'legal' ? (['nit'] as const) : allTypes;
  return types.map((value) => ({ value, label: documentTypeLabels[value] }));
}
