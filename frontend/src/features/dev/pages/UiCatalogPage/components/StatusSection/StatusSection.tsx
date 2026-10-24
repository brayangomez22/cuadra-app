import { Alert, Badge, IconAlert, Inline, Stack } from '@/shared/ui';
import { Section } from '../Section/Section';

/** Badge and Alert in each tone. */
export function StatusSection() {
  return (
    <>
      <Section title="Badge">
        <Inline gap="3" wrap>
          <Badge>Borrador</Badge>
          <Badge tone="success">Pagada</Badge>
          <Badge tone="warning">Stock bajo</Badge>
          <Badge tone="danger" icon={<IconAlert />}>
            Rechazada por la DIAN
          </Badge>
          <Badge tone="info">En proceso</Badge>
        </Inline>
      </Section>

      <Section title="Alert">
        <Stack gap="2">
          <Alert tone="danger">El correo o la contraseña no son correctos.</Alert>
          <Alert tone="warning" title="Sin conexión">
            No pudimos guardar los cambios. Revise el internet e intente de nuevo.
          </Alert>
          <Alert tone="info">La factura electrónica se enviará a la DIAN en unos minutos.</Alert>
          <Alert tone="success">Entrada de mercancía registrada.</Alert>
        </Stack>
      </Section>
    </>
  );
}
