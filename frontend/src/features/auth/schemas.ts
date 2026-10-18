import { z } from 'zod';

// Same limits as api/openapi.yaml. The domain rules (NIT check digit, email
// normalization) are checked by the API, whose 422 is shown on the field.

const email = z
  .string()
  .trim()
  .min(1, 'Escriba su correo')
  .max(254, 'El correo puede tener máximo 254 caracteres')
  .pipe(z.email('Escriba un correo válido, como ana@minegocio.co'));

export const loginSchema = z.object({
  email,
  password: z.string().min(1, 'Escriba su contraseña').max(72, 'La contraseña puede tener máximo 72 caracteres'),
});

export type LoginValues = z.infer<typeof loginSchema>;

export const signUpSchema = z.object({
  tenant_name: z
    .string()
    .trim()
    .min(1, 'Escriba el nombre de la empresa')
    .max(200, 'El nombre puede tener máximo 200 caracteres'),
  // Digits with optional dots and spaces, and the check digit with or without a dash.
  nit: z
    .string()
    .trim()
    .min(1, 'Escriba el NIT')
    .max(20, 'El NIT puede tener máximo 20 caracteres')
    .regex(/^\d[\d. ]*(-\d)?$/, 'Escriba el NIT con su dígito de verificación, por ejemplo 900123456-8'),
  name: z.string().trim().min(1, 'Escriba su nombre').max(200, 'El nombre puede tener máximo 200 caracteres'),
  email,
  password: z
    .string()
    .min(8, 'La contraseña debe tener al menos 8 caracteres')
    .max(72, 'La contraseña puede tener máximo 72 caracteres'),
  // An unchecked radio group gives null.
  business_template: z.string('Elija el tipo de negocio').min(1, 'Elija el tipo de negocio'),
});

export type SignUpValues = z.infer<typeof signUpSchema>;
