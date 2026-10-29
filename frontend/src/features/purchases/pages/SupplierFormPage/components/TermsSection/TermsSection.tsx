import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Checkbox, Field, FormGrid, FormSection, Input } from '@/shared/ui';
import type { SupplierInputValues } from '../../../../model/supplierForm/supplierSchema';

/** "Condiciones": the days of credit and, once it exists, whether it is active. */
export function TermsSection({ editing }: { editing: boolean }) {
  const { control, register } = useFormContext<SupplierInputValues>();
  const errorOf = useFieldError(control);
  return (
    <FormSection title="Condiciones">
      <FormGrid>
        <Field
          label="Plazo de pago"
          hint="Días de crédito que le da; con 0, le paga de contado. Propone el vencimiento de sus compras."
          error={errorOf('payment_term_days')}
        >
          <Input inputMode="numeric" align="end" autoComplete="off" {...register('payment_term_days')} />
        </Field>
      </FormGrid>
      {editing && (
        <Checkbox
          label="Activo"
          hint="A un proveedor inactivo no se le registran compras; las que tiene lo siguen nombrando."
          {...register('active')}
        />
      )}
    </FormSection>
  );
}
