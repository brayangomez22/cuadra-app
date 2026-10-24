import { useState } from 'react';
import { describeError } from '@/shared/api/errors';
import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';

type Props = {
  /** What is deleted, as the title asks: "¿Borrar la unidad Metro?". */
  title: string;
  description: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Deletes; a failure stays in the dialog with the API's message. */
  onConfirm: () => Promise<void>;
};

/** Asks before deleting something of the settings, and says why it could not. */
export function ConfirmDeleteDialog({ title, description, open, onOpenChange, onConfirm }: Props) {
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function confirm() {
    setError(null);
    setBusy(true);
    try {
      await onConfirm();
      onOpenChange(false);
    } catch (err) {
      setError(describeError(err).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) setError(null);
        onOpenChange(next);
      }}
      title={title}
      description={description}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button variant="danger" loading={busy} onClick={() => void confirm()}>
            Borrar
          </Button>
        </>
      }
    >
      {error && <Alert tone="danger">{error}</Alert>}
    </Dialog>
  );
}
