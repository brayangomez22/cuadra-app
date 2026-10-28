import { Controller } from 'react-hook-form';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Alert, Button, Field, FileInput, type SelectOption } from '@/shared/ui';
import type { ImportPreview } from '../../../../api/types';
import { useUploadForm } from '../../hooks/useUploadForm';
import { TemplateDownloads } from '../TemplateDownloads/TemplateDownloads';
import styles from './UploadForm.module.scss';

type Props = { locations: SelectOption[]; onUploaded: (preview: ImportPreview) => void };

const policies: SelectOption[] = [
  { value: 'update', label: 'Actualizar el producto' },
  { value: 'skip', label: 'Dejar el producto como está' },
];

/** The template, the location of the initial stock, what to do with known SKUs and the file to check. */
export function UploadForm(props: Props) {
  const { form, formError, submit, isPending } = useUploadForm(props);
  const errorOf = useFieldError(form.control);

  return (
    <form className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
      <TemplateDownloads />
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Field label="Sede del stock inicial" error={errorOf('location_id')}>
        <SelectController
          control={form.control}
          name="location_id"
          options={props.locations}
          placeholder="Elija la sede"
        />
      </Field>
      <Field
        label="Si el SKU ya existe"
        hint="Actualizar reemplaza sus datos y agrega las variantes nuevas. El stock inicial solo entra a lo que se crea."
      >
        <SelectController control={form.control} name="on_existing_sku" options={policies} />
      </Field>
      <Controller
        control={form.control}
        name="file"
        render={({ field, fieldState }) => (
          <Field
            label="Archivo"
            hint="La plantilla llena, en Excel (.xlsx) o CSV, hasta 10 MB."
            error={fieldState.error?.message}
          >
            <FileInput
              accept=".xlsx,.csv"
              name={field.name}
              ref={field.ref}
              onBlur={field.onBlur}
              onFile={field.onChange}
            />
          </Field>
        )}
      />
      <div className={styles.footer}>
        <Button type="submit" size="lg" loading={isPending}>
          Revisar archivo
        </Button>
      </div>
    </form>
  );
}
