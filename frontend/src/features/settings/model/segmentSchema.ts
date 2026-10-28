import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

export const segmentSchema = z.object({
  name: z.string().trim().min(1, 'Escriba el nombre').max(60, 'El nombre puede tener máximo 60 caracteres'),
});

export type SegmentValues = z.infer<typeof segmentSchema>;

// The field each 4xx code of /customer-segments belongs to (api/openapi.yaml).
export const segmentErrorFields: FieldsByErrorCode<SegmentValues> = {
  customer_segment_name_taken: 'name',
  invalid_customer_segment_name: 'name',
};
