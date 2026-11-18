import { z } from 'zod';

// Same limits as api/openapi.yaml; the API also rejects the most used
// passwords (password_too_common), shown on the field.
const newPassword = z
  .string()
  .min(8, 'La contraseña debe tener al menos 8 caracteres')
  .max(72, 'La contraseña puede tener máximo 72 caracteres');

export const acceptInvitationSchema = z.object({
  name: z.string().trim().min(1, 'Escriba su nombre').max(200, 'El nombre puede tener máximo 200 caracteres'),
  password: newPassword,
});

export type AcceptInvitationValues = z.infer<typeof acceptInvitationSchema>;

export const resetPasswordSchema = z.object({ password: newPassword });

export type ResetPasswordValues = z.infer<typeof resetPasswordSchema>;

export const changePasswordSchema = z.object({
  current_password: z.string().min(1, 'Escriba su contraseña actual').max(72, 'Máximo 72 caracteres'),
  new_password: newPassword,
});

export type ChangePasswordValues = z.infer<typeof changePasswordSchema>;
