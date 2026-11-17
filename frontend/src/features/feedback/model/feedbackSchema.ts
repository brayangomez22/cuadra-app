import { z } from 'zod';

/** The comment a user sends from the app. */
export const feedbackSchema = z.object({
  message: z
    .string()
    .trim()
    .min(1, { error: 'Escriba su comentario.' })
    .max(2000, { error: 'Máximo 2000 caracteres.' }),
});

export type FeedbackValues = z.infer<typeof feedbackSchema>;
