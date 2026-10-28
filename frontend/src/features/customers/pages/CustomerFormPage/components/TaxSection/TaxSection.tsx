import { Controller, useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Checkbox, FormSection, IconAlert } from '@/shared/ui';
import type { CustomerSegment } from '../../../../api/types';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';
import { taxResponsibilityOptions, toggleResponsibility } from '../../../../model/customerForm/taxResponsibilities';
import { SegmentSelect } from '../SegmentSelect/SegmentSelect';
import styles from './TaxSection.module.scss';

type Props = { segments: CustomerSegment[]; editing: boolean };

/** "Clasificación": the segment, the RUT's responsibilities and, once it exists, whether it is active. */
export function TaxSection({ segments, editing }: Props) {
  const { control, register } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control, 'tax_responsibilities');
  const error = errorOf('tax_responsibilities');
  return (
    <FormSection title="Clasificación">
      <SegmentSelect segments={segments} />
      <fieldset className={styles.responsibilities} aria-invalid={error ? true : undefined}>
        <legend className={styles.legend}>Responsabilidades fiscales</legend>
        <p className={styles.hint}>Las del RUT del cliente. Si no tiene ninguna de estas, «No responsable».</p>
        <Controller
          control={control}
          name="tax_responsibilities"
          render={({ field }) => (
            <div className={styles.options}>
              {taxResponsibilityOptions.map((option) => (
                <Checkbox
                  key={option.value}
                  label={option.label}
                  checked={field.value.includes(option.value)}
                  onChange={(event) =>
                    field.onChange(toggleResponsibility(field.value, option.value, event.target.checked))
                  }
                />
              ))}
            </div>
          )}
        />
        {error && (
          <p className={styles.error} role="alert">
            <IconAlert className={styles.errorIcon} />
            <span>{error}</span>
          </p>
        )}
      </fieldset>
      {editing && (
        <Checkbox
          label="Activo"
          hint="A un cliente inactivo no se le puede vender, y no aparece en la búsqueda del POS."
          {...register('active')}
        />
      )}
    </FormSection>
  );
}
