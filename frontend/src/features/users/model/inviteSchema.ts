import { z } from 'zod';

// Same limits as api/openapi.yaml (InvitationRequest).
export const inviteSchema = z.object({
  email: z
    .string()
    .trim()
    .min(1, 'Escriba el correo')
    .max(254, 'El correo puede tener máximo 254 caracteres')
    .pipe(z.email('Escriba un correo válido, como luis@minegocio.co')),
  name: z.string().trim().min(1, 'Escriba el nombre').max(200, 'El nombre puede tener máximo 200 caracteres'),
  role: z.enum(['owner', 'admin', 'cashier', 'warehouse'], 'Elija el rol'),
});

export type InviteValues = z.infer<typeof inviteSchema>;

export const roleSchema = z.object({ role: inviteSchema.shape.role });

export type RoleValues = z.infer<typeof roleSchema>;
