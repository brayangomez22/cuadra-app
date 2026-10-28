import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import type { SelectOption } from '@/shared/ui';
import { useUploadImport } from '../../../api/imports';
import type { ImportPreview } from '../../../api/types';
import {
  importErrorFields,
  importSchema,
  type ImportFormInput,
  type ImportFormValues,
} from '../../../model/importSchema';

type Input = { locations: SelectOption[]; onUploaded: (preview: ImportPreview) => void };

/** The upload form: the location (chosen already when there is one), what to do with known SKUs and the file. */
export function useUploadForm({ locations, onUploaded }: Input) {
  const upload = useUploadImport();
  const form = useForm<ImportFormInput, unknown, ImportFormValues>({
    resolver: zodResolver(importSchema),
    defaultValues: {
      location_id: locations.length === 1 ? (locations[0]?.value ?? '') : '',
      on_existing_sku: 'update',
      file: null,
    },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, importErrorFields);

  const submit = form.handleSubmit(({ file, ...fields }) => {
    clearFormError();
    if (!file) return;
    upload.mutate({ file, ...fields }, { onSuccess: onUploaded, onError: handleError });
  });

  return { form, formError, submit, isPending: upload.isPending };
}
