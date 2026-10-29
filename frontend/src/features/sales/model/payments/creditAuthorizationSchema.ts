import { z } from 'zod';

/** The email and password of an administrator who authorizes a credit sale over the customer's limit. */
export const creditAuthorizationSchema = z.object({
  email: z.string().trim().min(1, 'Escriba el correo del administrador').max(254),
  password: z.string().min(1, 'Escriba la contraseña').max(128),
});

export type CreditAuthorizationValues = z.input<typeof creditAuthorizationSchema>;
