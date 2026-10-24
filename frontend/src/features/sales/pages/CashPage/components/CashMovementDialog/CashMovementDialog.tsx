import { useId } from 'react';
import { Button, Dialog, DialogClose } from '@/shared/ui';
import type { CashMovementKind } from '../../../../model/cashLabels';
import { MovementForm } from './MovementForm';

const copy: Record<CashMovementKind, { title: string; description: string; reason: string }> = {
  in: {
    title: 'Registrar ingreso',
    description: 'Efectivo que entra al cajón sin ser una venta, como más sencillo.',
    reason: 'Por ejemplo: sencillo que trajo el dueño',
  },
  out: {
    title: 'Registrar egreso',
    description: 'Efectivo que sale del cajón, como un pago o un retiro.',
    reason: 'Por ejemplo: pago del domicilio',
  },
};

type Props = {
  sessionId: string;
  /** The movement being recorded; without it the dialog is closed. */
  kind?: CashMovementKind;
  onOpenChange: (open: boolean) => void;
};

/** A cash in or out of the open session, with its reason. */
export function CashMovementDialog({ sessionId, kind, onOpenChange }: Props) {
  const text = copy[kind ?? 'in'];
  const formId = useId();
  return (
    <Dialog
      open={kind !== undefined}
      onOpenChange={onOpenChange}
      title={text.title}
      description={text.description}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId}>
            {text.title}
          </Button>
        </>
      }
    >
      {/* Mounted per opening: every movement starts empty. */}
      {kind && (
        <MovementForm
          id={formId}
          sessionId={sessionId}
          kind={kind}
          reasonHint={text.reason}
          onDone={() => onOpenChange(false)}
        />
      )}
    </Dialog>
  );
}
