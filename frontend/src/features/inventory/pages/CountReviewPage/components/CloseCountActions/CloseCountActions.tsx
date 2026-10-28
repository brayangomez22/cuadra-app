import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';
import { useCloseCount } from '../../hooks/useCloseCount';
import { ApproveCountDialog } from '../ApproveCountDialog/ApproveCountDialog';

type Props = {
  countId: string;
  /** The name of a variant of the count, if its row is loaded. */
  nameOf: (variantId: string) => string | undefined;
  /** A product short of stock was sent to recount: the screen takes the user to its line. */
  onRecounted: (variantId: string) => void;
};

/** Cancel the count, or approve it and adjust the stock; both ask first. */
export function CloseCountActions({ countId, nameOf, onRecounted }: Props) {
  const close = useCloseCount(countId);
  return (
    <>
      <ApproveCountDialog countId={countId} close={close} nameOf={nameOf} onRecounted={onRecounted} />
      <Dialog
        onOpenChange={(open) => !open && close.clearError()}
        trigger={<Button variant="secondary">Cancelar conteo</Button>}
        title="Cancelar el conteo"
        description="El conteo se cierra sin ajustar el inventario. Lo contado queda guardado, pero no cuenta."
        footer={
          <>
            <DialogClose asChild>
              <Button variant="ghost">Volver</Button>
            </DialogClose>
            <Button variant="danger" loading={close.isCanceling} onClick={close.cancel}>
              Cancelar conteo
            </Button>
          </>
        }
      >
        {close.error && <Alert tone="danger">{close.error}</Alert>}
      </Dialog>
    </>
  );
}
