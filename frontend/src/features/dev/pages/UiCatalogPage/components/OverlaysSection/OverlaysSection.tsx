import {
  Button,
  Dialog,
  DialogClose,
  Drawer,
  EmptyState,
  Field,
  IconBox,
  IconButton,
  IconMenu,
  Inline,
  Input,
} from '@/shared/ui';
import { FormGrid } from '../FormGrid/FormGrid';
import { Section } from '../Section/Section';

// A confirmation that destroys something: danger, with a reason.
function VoidSaleDialog() {
  return (
    <Dialog
      trigger={<Button variant="danger">Anular venta</Button>}
      title="¿Anular la venta 1024?"
      description="La venta queda anulada y los productos vuelven al inventario. Esta acción no se puede deshacer."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <DialogClose asChild>
            <Button variant="danger">Anular venta</Button>
          </DialogClose>
        </>
      }
    >
      <Field label="Motivo" hint="Queda en la auditoría">
        <Input />
      </Field>
    </Dialog>
  );
}

// A form in a medium dialog.
function ReceiptDialog() {
  return (
    <Dialog
      size="md"
      trigger={<Button variant="secondary">Registrar entrada</Button>}
      title="Registrar entrada de mercancía"
      description="Las cantidades se suman al inventario de la sede Principal."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <Button>Registrar entrada</Button>
        </>
      }
    >
      <FormGrid>
        <Field label="Producto">
          <Input placeholder="Buscar por nombre o código" />
        </Field>
        <Field label="Cantidad">
          <Input inputMode="decimal" align="end" />
        </Field>
        <Field label="Costo unitario">
          <Input inputMode="decimal" align="end" />
        </Field>
      </FormGrid>
    </Dialog>
  );
}

/** Drawer and Dialog, opened from their triggers. */
export function OverlaysSection() {
  return (
    <>
      <Section title="Drawer">
        <Inline gap="3" wrap>
          <Drawer title="Menú" trigger={<IconButton label="Abrir menú" icon={<IconMenu />} variant="secondary" />}>
            <EmptyState
              icon={<IconBox />}
              title="Contenido del panel"
              description="En la app, aquí van el menú y la cuenta."
              headingLevel={3}
            />
          </Drawer>
        </Inline>
      </Section>

      <Section title="Dialog">
        <Inline gap="3" wrap>
          <VoidSaleDialog />
          <ReceiptDialog />
        </Inline>
      </Section>
    </>
  );
}
