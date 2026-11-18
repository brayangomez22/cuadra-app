import { useId, type ComponentProps, type ReactNode } from 'react';
import { Alert, Button, Dialog, DialogClose, Stack } from '@/shared/ui';

type Props = {
  open: boolean;
  /** Closing by Cancelar, Escape or the overlay. */
  onClose: () => void;
  title: string;
  description: string;
  size?: 'sm' | 'md';
  onSubmit: ComponentProps<'form'>['onSubmit'];
  isSubmitting: boolean;
  /** The message of an error no field takes (useSaveDialog). */
  formError: string | null;
  /** The form's fields. */
  children: ReactNode;
};

/** The create or edit dialog of a settings list: its form, Cancelar and Guardar. */
export function FormDialog({ open, onClose, onSubmit, isSubmitting, formError, children, ...dialog }: Props) {
  const formId = useId();
  return (
    <Dialog
      {...dialog}
      open={open}
      onOpenChange={(next) => !next && onClose()}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={isSubmitting}>
            Guardar
          </Button>
        </>
      }
    >
      <form id={formId} noValidate onSubmit={onSubmit}>
        <Stack>
          {formError && <Alert tone="danger">{formError}</Alert>}
          {children}
        </Stack>
      </form>
    </Dialog>
  );
}
