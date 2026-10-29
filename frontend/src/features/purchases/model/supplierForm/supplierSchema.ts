import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

// Same limits as SupplierInput in api/openapi.yaml. The NIT's check digit is
// the API's: its 422 shows on the NIT.
const optional = (max: number, label: string) =>
  z
    .string()
    .trim()
    .max(max, `${label} puede tener máximo ${String(max)} caracteres`);

const TERM_MESSAGE = 'El plazo va de 0 a 365 días';

export const supplierSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Escriba el nombre del proveedor')
    .max(200, 'El nombre puede tener máximo 200 caracteres'),
  nit: z
    .string()
    .trim()
    .min(1, 'Escriba el NIT con su dígito de verificación')
    .max(30, 'El NIT puede tener máximo 30 caracteres'),
  contact_name: optional(200, 'El contacto'),
  email: optional(254, 'El correo').refine((v) => v === '' || z.email().safeParse(v).success, {
    message: 'Escriba un correo válido',
  }),
  phone: optional(30, 'El teléfono'),
  // A text field: empty is cash (0 days).
  payment_term_days: z
    .string()
    .trim()
    .refine((text) => /^\d{0,3}$/.test(text) && Number(text) <= 365, TERM_MESSAGE)
    .transform(Number),
  active: z.boolean(),
});

export type SupplierInputValues = z.input<typeof supplierSchema>;
export type SupplierValues = z.output<typeof supplierSchema>;

/** The contact fields a supplier may leave empty: the API stores them as absent. */
export const optionalSupplierFields = ['contact_name', 'email', 'phone'] as const;

// The field each 4xx code of /suppliers belongs to (api/openapi.yaml).
export const supplierErrorFields: FieldsByErrorCode<SupplierInputValues> = {
  invalid_nit: 'nit',
  invalid_nit_check_digit: 'nit',
  supplier_nit_taken: 'nit',
  invalid_supplier_name: 'name',
  invalid_payment_term: 'payment_term_days',
};
