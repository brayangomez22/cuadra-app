import { Button, IconButton, IconClose, IconPlus, IconSearch, IconSuccess, Inline } from '@/shared/ui';
import { Section } from '../Section/Section';

/** Button and IconButton in their variants, sizes and states. */
export function ButtonsSection() {
  return (
    <>
      <Section title="Button">
        <Inline gap="3" wrap>
          <Button>Cobrar</Button>
          <Button variant="secondary">Guardar borrador</Button>
          <Button variant="ghost">Cancelar</Button>
          <Button variant="danger">Anular venta</Button>
        </Inline>
        <Inline gap="3" wrap>
          <Button size="lg" icon={<IconSuccess />}>
            Cobrar $ 128.400
          </Button>
          <Button icon={<IconPlus />} variant="secondary">
            Registrar entrada
          </Button>
          <Button loading>Guardando</Button>
          <Button disabled>Deshabilitado</Button>
          <Button variant="secondary" disabled>
            Deshabilitado
          </Button>
        </Inline>
      </Section>

      <Section title="IconButton">
        <Inline gap="3" wrap>
          <IconButton label="Buscar producto" icon={<IconSearch />} />
          <IconButton label="Agregar" icon={<IconPlus />} variant="secondary" />
          <IconButton label="Quitar producto" icon={<IconClose />} variant="danger" />
          <IconButton label="Deshabilitado" icon={<IconClose />} disabled />
        </Inline>
      </Section>
    </>
  );
}
