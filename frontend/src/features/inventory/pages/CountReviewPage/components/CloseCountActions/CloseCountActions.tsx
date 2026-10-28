import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';
import { useCloseCount } from '../../hooks/useCloseCount';

/** Cancel the count, or approve it and adjust the stock; both ask first. */
export function CloseCountActions({ countId }: { countId: string }) {
  const close = useCloseCount(countId);
  const reset = (open: boolean) => {
    if (!open) close.clearError();
  };
  const back = (
    <DialogClose asChild>
      <Button variant="ghost">Volver</Button>
    </DialogClose>
  );

  return (
    <>
      <Dialog
        onOpenChange={reset}
        trigger={<Button>Aprobar conteo</Button>}
        title="Aprobar el conteo"
        description="Se registra un ajuste «Conteo físico» por cada producto con diferencia. Lo que no se contó no se ajusta."
        footer={
          <>
            {back}
            <Button loading={close.isApproving} onClick={close.approve}>
              Aprobar y ajustar
            </Button>
          </>
        }
      >
        {close.error && <Alert tone="danger">{close.error}</Alert>}
      </Dialog>
      <Dialog
        onOpenChange={reset}
        trigger={<Button variant="secondary">Cancelar conteo</Button>}
        title="Cancelar el conteo"
        description="El conteo se cierra sin ajustar el inventario. Lo contado queda guardado, pero no cuenta."
        footer={
          <>
            {back}
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
