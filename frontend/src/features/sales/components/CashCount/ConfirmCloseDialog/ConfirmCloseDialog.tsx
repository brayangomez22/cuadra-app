import { describeError } from '@/shared/api/errors';
import { formatMoney } from '@/shared/format';
import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The cash counted, to read once more before closing. */
  total: string;
  onConfirm: () => void;
  /** The closing request: its progress and its error. */
  closing: { isPending: boolean; isError: boolean; error: unknown };
};

/** Asks before closing the drawer: after that it cannot be changed. */
export function ConfirmCloseDialog({ open, onOpenChange, total, onConfirm, closing }: Props) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="¿Cerrar la caja?"
      description={`Contó ${formatMoney(total)}. Después de cerrarla no se puede modificar.`}
      footer={
        <>
          {/* The safe choice has the focus: closing cannot be undone. */}
          <DialogClose asChild>
            <Button variant="secondary" autoFocus>
              Seguir contando
            </Button>
          </DialogClose>
          <Button onClick={onConfirm} loading={closing.isPending}>
            Cerrar caja
          </Button>
        </>
      }
    >
      {closing.isError && <Alert tone="danger">{describeError(closing.error).message}</Alert>}
    </Dialog>
  );
}
