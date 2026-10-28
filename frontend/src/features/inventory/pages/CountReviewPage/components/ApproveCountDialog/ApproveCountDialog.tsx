import { useState } from 'react';
import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';
import type { CloseCount } from '../../hooks/useCloseCount';
import { ShortStockAlert } from '../ShortStockAlert/ShortStockAlert';

type Props = {
  countId: string;
  close: CloseCount;
  /** The name of a variant of the count, if its row is loaded. */
  nameOf: (variantId: string) => string | undefined;
  /** A product short of stock was sent to recount: the screen takes the user to its line. */
  onRecounted: (variantId: string) => void;
};

/** Approves the count after asking; a product left without stock can be sent to recount from here. */
export function ApproveCountDialog({ countId, close, nameOf, onRecounted }: Props) {
  const [open, setOpen] = useState(false);
  const change = (next: boolean) => {
    setOpen(next);
    if (!next) close.clearError();
  };
  return (
    <Dialog
      open={open}
      onOpenChange={change}
      trigger={<Button>Aprobar conteo</Button>}
      title="Aprobar el conteo"
      description="Se registra un ajuste «Conteo físico» por cada producto con diferencia. Lo que no se contó no se ajusta."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Volver</Button>
          </DialogClose>
          <Button loading={close.isApproving} onClick={close.approve}>
            Aprobar y ajustar
          </Button>
        </>
      }
    >
      {close.short ? (
        <ShortStockAlert
          countId={countId}
          variantId={close.short}
          name={nameOf(close.short)}
          onRecounted={(variantId) => {
            change(false);
            onRecounted(variantId);
          }}
        />
      ) : (
        close.error && <Alert tone="danger">{close.error}</Alert>
      )}
    </Dialog>
  );
}
