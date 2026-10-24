import { useId, useState, type ReactElement } from 'react';
import { Alert, Button, Dialog, DialogClose, Field, Input } from '@/shared/ui';
import type { Location } from '../../api/types';
import styles from './CreateLocationDialog.module.scss';
import { useCreateLocationForm } from '../../hooks/useCreateLocationForm';

type Props = {
  /** The button that opens the dialog. */
  trigger: ReactElement;
  onCreated: (location: Location) => void;
};

/** Creates a location (sede o bodega). Only the owner can (tenant:manage). */
export function CreateLocationDialog({ trigger, onCreated }: Props) {
  const formId = useId();
  const [open, setOpen] = useState(false);
  const { form, formError, submit, discard, isPending } = useCreateLocationForm((location) => {
    onCreated(location);
    setOpen(false);
  });

  function changeOpen(next: boolean) {
    setOpen(next);
    if (!next) discard();
  }

  return (
    <Dialog
      open={open}
      onOpenChange={changeOpen}
      trigger={trigger}
      title="Nueva sede"
      description="Un local o una bodega donde guarda mercancía. Cada sede lleva su propia existencia."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={isPending}>
            Crear sede
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        <Field label="Nombre de la sede" error={form.formState.errors.name?.message}>
          <Input autoComplete="off" placeholder="Bodega principal" {...form.register('name')} />
        </Field>
      </form>
    </Dialog>
  );
}
