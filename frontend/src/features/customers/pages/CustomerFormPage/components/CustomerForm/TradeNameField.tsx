import { useFormContext } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Field, Input } from '@/shared/ui';
import type { CustomerValues } from '../../../../model/customerForm/customerSchema';

/** The trade name, when the customer is known by another name than its legal one. */
export function TradeNameField() {
  const { control, register } = useFormContext<CustomerValues>();
  const errorOf = useFieldError(control, 'trade_name');
  return (
    <Field label="Nombre comercial" optional hint="Si se conoce por otro nombre." error={errorOf('trade_name')}>
      <Input autoComplete="off" {...register('trade_name')} />
    </Field>
  );
}
