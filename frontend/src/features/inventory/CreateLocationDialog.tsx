import { zodResolver } from '@hookform/resolvers/zod';
import { useQueryClient } from '@tanstack/react-query';
import { useId, useState, type ReactElement } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { api } from '@/shared/api/client';
import { describeError, unwrap } from '@/shared/api/errors';
import { Alert, Button, Dialog, DialogClose, Field, Input, useToast } from '@/shared/ui';
import { inventoryKeys, type Location } from './api';
import styles from './CreateLocationDialog.module.scss';

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Escriba el nombre de la sede')
    .max(100, 'El nombre puede tener máximo 100 caracteres'),
});

type Values = z.infer<typeof schema>;

type Props = {
  /** The button that opens the dialog. */
  trigger: ReactElement;
  onCreated: (location: Location) => void;
};

/** Creates a location (sede o bodega). Only the owner can (tenant:manage). */
export function CreateLocationDialog({ trigger, onCreated }: Props) {
  const formId = useId();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const [open, setOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm({ resolver: zodResolver(schema), defaultValues: { name: '' } });

  async function create(values: Values) {
    setFormError(null);
    try {
      const location = unwrap(await api.POST('/api/v1/locations', { body: values }));
      await queryClient.invalidateQueries({ queryKey: inventoryKeys.locations });
      toast({ tone: 'success', title: 'Sede creada', description: location.name });
      onCreated(location);
      setOpen(false);
      reset();
    } catch (error) {
      const { code, message } = describeError(error);
      if (code === 'location_name_taken' || code === 'invalid_location_name') {
        setError('name', { message }, { shouldFocus: true });
      } else {
        setFormError(message);
      }
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) {
          reset();
          setFormError(null);
        }
      }}
      trigger={trigger}
      title="Nueva sede"
      description="Un local o una bodega donde guarda mercancía. Cada sede lleva su propia existencia."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={isSubmitting}>
            Crear sede
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.form} noValidate onSubmit={handleSubmit(create)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Nombre de la sede" error={errors.name?.message}>
          <Input autoComplete="off" placeholder="Bodega principal" {...register('name')} />
        </Field>
      </form>
    </Dialog>
  );
}
