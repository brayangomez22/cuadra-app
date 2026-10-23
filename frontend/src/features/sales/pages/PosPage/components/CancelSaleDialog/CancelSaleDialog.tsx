import { Button, Dialog, DialogClose } from '@/shared/ui';

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Empties the cart. */
  onCancelSale: () => void;
};

/** Asks before Escape empties the cart: nothing of the sale is kept. */
export function CancelSaleDialog({ open, onOpenChange, onCancelSale }: Props) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="¿Cancelar la venta?"
      description="Se quitan todos los productos del carrito. No se guarda nada."
      footer={
        <>
          {/* The safe choice has the focus: the Enter of a scan keeps the sale. */}
          <DialogClose asChild>
            <Button variant="secondary" autoFocus>
              Seguir vendiendo
            </Button>
          </DialogClose>
          <Button variant="danger" onClick={onCancelSale}>
            Cancelar venta
          </Button>
        </>
      }
    />
  );
}
