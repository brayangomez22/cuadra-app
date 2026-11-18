import { Button, Dialog, DialogClose } from '@/shared/ui';
import type { User } from '../../../../api/types';

type Props = { user: User | null; deactivating: boolean; onConfirm: (user: User) => void; onClose: () => void };

/** Confirms a deactivation: the user loses access at once, and nothing of theirs is deleted. */
export function DeactivateDialog({ user, deactivating, onConfirm, onClose }: Props) {
  return (
    <Dialog
      open={user !== null}
      onOpenChange={(open) => !open && onClose()}
      title={`¿Desactivar a ${user?.name ?? ''}?`}
      description="Pierde el acceso de inmediato. Sus ventas y movimientos se conservan, y puede reactivarlo cuando quiera."
      size="sm"
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button variant="danger" loading={deactivating} onClick={() => user && onConfirm(user)}>
            Desactivar
          </Button>
        </>
      }
    />
  );
}
