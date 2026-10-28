import { zodResolver } from '@hookform/resolvers/zod';
import { useId } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { Alert, Button, Dialog, DialogClose, Field, Input, Stack } from '@/shared/ui';
import { useSaveCustomer } from '../../api/customers';
import type { Customer } from '../../api/types';
import { quickCustomerDefaults } from '../../model/chosenCustomer';
import { customerErrorFields, customerSchema } from '../../model/customerForm/customerSchema';
import { toCustomerInput } from '../../model/customerForm/toCustomerInput';
import { IdentityFields } from '../IdentityFields/IdentityFields';

type Props = {
  /** What the cashier searched: the document's number or the name to start with. */
  searched: string;
  onClose: () => void;
  onCreated: (customer: Customer) => void;
};

/** Creates a customer from the POS with the least it needs; the rest is completed at /clientes. */
export function QuickCustomerDialog({ searched, onClose, onCreated }: Props) {
  const formId = useId();
  const defaults = quickCustomerDefaults(searched);
  const form = useForm({ resolver: zodResolver(customerSchema), defaultValues: defaults });
  const { mutateAsync } = useSaveCustomer();
  const { formError, handleError } = useApiFormErrors(form.setError, customerErrorFields);
  const submit = form.handleSubmit(async (values) => {
    try {
      onCreated(await mutateAsync(toCustomerInput(values)));
    } catch (error) {
      handleError(error);
    }
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title="Nuevo cliente"
      description="Lo mínimo para venderle y facturarle. El correo y lo demás se completan en Clientes."
      size="md"
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={form.formState.isSubmitting}>
            Guardar
          </Button>
        </>
      }
    >
      <FormProvider {...form}>
        <form id={formId} noValidate onSubmit={(event) => void submit(event)}>
          <Stack>
            {formError && <Alert tone="danger">{formError}</Alert>}
            {/* The cashier goes on with the field the search did not fill. */}
            <IdentityFields autoFocus={defaults.document_number ? 'name' : 'document_number'} />
            <Field label="Teléfono" optional error={form.formState.errors.phone?.message}>
              <Input type="tel" autoComplete="off" {...form.register('phone')} />
            </Field>
          </Stack>
        </form>
      </FormProvider>
    </Dialog>
  );
}
