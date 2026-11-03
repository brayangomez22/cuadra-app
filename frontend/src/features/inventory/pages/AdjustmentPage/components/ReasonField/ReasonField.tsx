import { useFormContext, useWatch } from 'react-hook-form';
import { useFieldError } from '@/shared/forms/useFieldError';
import { Button, Field, Inline, Input, Stack } from '@/shared/ui';
import { REASONS, type AdjustmentFormInput } from '../../../../model/stockChange/adjustmentSchema';

/** The reason, which goes to the kardex: free text, with the usual ones of the chosen direction as shortcuts. */
export function ReasonField() {
  const { control, register, setValue } = useFormContext<AdjustmentFormInput>();
  const direction = useWatch({ control, name: 'direction' });
  const errorOf = useFieldError(control, 'reason');
  return (
    <Stack gap="2">
      <Field label="Motivo" error={errorOf('reason')}>
        <Input autoComplete="off" {...register('reason')} />
      </Field>
      {direction && (
        <Inline gap="2" wrap>
          {REASONS[direction].map((reason) => (
            <Button
              key={reason}
              variant="secondary"
              onClick={() => setValue('reason', reason, { shouldValidate: true })}
            >
              {reason}
            </Button>
          ))}
        </Inline>
      )}
    </Stack>
  );
}
