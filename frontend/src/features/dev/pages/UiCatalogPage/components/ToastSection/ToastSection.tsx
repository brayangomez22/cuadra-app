import { Button, Inline, useToast } from '@/shared/ui';
import { Section } from '../Section/Section';

/** A button per Toast tone. */
export function ToastSection() {
  const { toast } = useToast();
  return (
    <Section title="Toast">
      <Inline gap="3" wrap>
        <Button
          variant="secondary"
          onClick={() =>
            toast({ tone: 'success', title: 'Venta guardada', description: 'Factura FE-1024 por $ 128.400' })
          }
        >
          Éxito
        </Button>
        <Button
          variant="secondary"
          onClick={() =>
            toast({
              tone: 'danger',
              title: 'No se pudo guardar la venta',
              description: 'No hay stock suficiente de Tenis de lona blancos (38): quedan 0.',
            })
          }
        >
          Error
        </Button>
        <Button
          variant="secondary"
          onClick={() =>
            toast({ tone: 'warning', title: 'Sin conexión', description: 'La venta se enviará al volver la conexión.' })
          }
        >
          Advertencia
        </Button>
        <Button variant="secondary" onClick={() => toast({ tone: 'info', title: 'Precios actualizados' })}>
          Información
        </Button>
      </Inline>
    </Section>
  );
}
