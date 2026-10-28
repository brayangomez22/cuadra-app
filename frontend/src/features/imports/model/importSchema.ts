import { z } from 'zod';
import type { FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';

// Same limit as ImportUpload in api/openapi.yaml: the API answers 413 above it.
export const MAX_FILE_BYTES = 10 * 1024 * 1024;

const supported = /\.(xlsx|csv)$/i;

// fileError says what is wrong with the chosen file, one thing at a time.
function fileError(file: File | null): string | undefined {
  if (!file) return 'Elija el archivo';
  if (!supported.test(file.name)) return 'El archivo debe ser .xlsx o .csv';
  if (file.size > MAX_FILE_BYTES) return 'El archivo pasa de 10 MB';
  return undefined;
}

export const importSchema = z.object({
  location_id: z.string().min(1, 'Elija la sede'),
  on_existing_sku: z.enum(['update', 'skip']),
  file: z.custom<File | null>().superRefine((file, ctx) => {
    const message = fileError(file);
    if (message) ctx.addIssue({ code: 'custom', message });
  }),
});

export type ImportFormInput = z.input<typeof importSchema>;
export type ImportFormValues = z.output<typeof importSchema>;

// Error codes of POST /imports (api/openapi.yaml) that belong to one field.
export const importErrorFields: FieldsByErrorCode<ImportFormInput> = {
  location_not_found: 'location_id',
  unsupported_format: 'file',
  unreadable_file: 'file',
  file_too_large: 'file',
};
