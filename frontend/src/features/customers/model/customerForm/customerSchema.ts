import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { NONE } from './taxResponsibilities';

// Same limits as api/openapi.yaml. The NIT's check digit and the canonical
// form of each document are the API's: its 422 shows on the number.
const optional = (max: number, label: string) =>
  z
    .string()
    .trim()
    .max(max, `${label} puede tener máximo ${String(max)} caracteres`);

export const customerSchema = z
  .object({
    person_type: z.enum(['natural', 'legal']),
    document_type: z.enum(['cc', 'nit', 'ce', 'passport']),
    document_number: z
      .string()
      .trim()
      .min(1, 'Escriba el número de documento')
      .max(30, 'El número puede tener máximo 30 caracteres'),
    name: z.string().trim().min(1, 'Escriba el nombre').max(200, 'El nombre puede tener máximo 200 caracteres'),
    trade_name: optional(200, 'El nombre comercial'),
    email: optional(254, 'El correo').refine((v) => v === '' || z.email().safeParse(v).success, {
      message: 'Escriba un correo válido',
    }),
    phone: optional(30, 'El teléfono'),
    address: optional(200, 'La dirección'),
    city: optional(100, 'La ciudad'),
    segment_id: z.string(),
    tax_responsibilities: z.array(z.enum(['O-13', 'O-15', 'O-23', 'O-47', 'R-99-PN'])),
    active: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.person_type === 'legal' && values.document_type !== 'nit') {
      ctx.addIssue({ code: 'custom', path: ['document_type'], message: 'Una persona jurídica se identifica con NIT' });
    }
    if (values.tax_responsibilities.includes(NONE) && values.tax_responsibilities.length > 1) {
      ctx.addIssue({
        code: 'custom',
        path: ['tax_responsibilities'],
        message: '«No responsable» no se combina con las demás',
      });
    }
  });

export type CustomerValues = z.infer<typeof customerSchema>;

/** The text fields a customer may leave empty: the API stores them as absent. */
export const optionalCustomerFields = ['trade_name', 'email', 'phone', 'address', 'city', 'segment_id'] as const;

// The field each 4xx code of /customers belongs to (api/openapi.yaml).
export const customerErrorFields: FieldsByErrorCode<CustomerValues> = {
  invalid_person_type: 'person_type',
  invalid_document: 'document_number',
  invalid_nit_check_digit: 'document_number',
  legal_person_requires_nit: 'document_type',
  customer_document_taken: 'document_number',
  invalid_customer_name: 'name',
  invalid_tax_responsibilities: 'tax_responsibilities',
  customer_segment_not_found: 'segment_id',
};
