import { useId, useState, type ReactNode } from 'react';
import {
  Badge,
  Button,
  Checkbox,
  Dialog,
  DialogClose,
  EmptyState,
  Field,
  IconAlert,
  IconBox,
  IconButton,
  IconClose,
  IconPlus,
  IconSearch,
  IconSuccess,
  Input,
  Select,
  Skeleton,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  useToast,
} from '@/shared/ui';
import styles from './UiCatalogPage.module.scss';

// Internal catalog of shared/ui in all its states. Only in development
// (router.tsx): it is the visual reference for reviews and screenshots.

const THEMES = [
  { value: 'system', label: 'Sistema' },
  { value: 'light', label: 'Claro' },
  { value: 'dark', label: 'Oscuro' },
];

const UNITS = [
  { value: 'und', label: 'Unidad' },
  { value: 'm', label: 'Metro' },
  { value: 'kg', label: 'Kilogramo' },
  { value: 'bulto', label: 'Bulto' },
  { value: 'rollo', label: 'Rollo', disabled: true },
];

const STOCK = [
  { sku: 'CEM-050', name: 'Cemento gris 50kg', unit: 'bulto', qty: '12', price: '$ 32.900', state: 'ok' },
  { sku: 'CAB-12', name: 'Cable eléctrico 12 AWG THHN rojo, rollo por metro', unit: 'm', qty: '2,5', price: '$ 3.450', state: 'low' },
  { sku: 'TOR-14', name: 'Tornillo drywall 6 × 1"', unit: 'und', qty: '1.240', price: '$ 90', state: 'ok' },
  { sku: 'PEG-PVC', name: 'Soldadura PVC 1/4 galón', unit: 'und', qty: '0', price: '$ 41.200', state: 'out' },
] as const;

function applyTheme(theme: string) {
  const root = document.documentElement;
  if (theme === 'system') delete root.dataset.theme;
  else root.dataset.theme = theme;
}

export function UiCatalogPage() {
  const { toast } = useToast();
  const [theme, setTheme] = useState('system');
  const [selectedSku, setSelectedSku] = useState<string>('CAB-12');

  return (
    <main className={styles.root}>
      <header className={styles.header}>
        <div>
          <h1 className={styles.title}>Componentes base</h1>
          <p className={styles.lead}>Catálogo interno de <code>shared/ui</code>. Solo existe en desarrollo.</p>
        </div>
        <Field label="Tema" className={styles.theme}>
          <Select
            options={THEMES}
            value={theme}
            onValueChange={(value) => {
              setTheme(value);
              applyTheme(value);
            }}
          />
        </Field>
      </header>

      <Section title="Button">
        <Row>
          <Button>Cobrar</Button>
          <Button variant="secondary">Guardar borrador</Button>
          <Button variant="ghost">Cancelar</Button>
          <Button variant="danger">Anular venta</Button>
        </Row>
        <Row>
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
        </Row>
      </Section>

      <Section title="IconButton">
        <Row>
          <IconButton label="Buscar producto" icon={<IconSearch />} />
          <IconButton label="Agregar" icon={<IconPlus />} variant="secondary" />
          <IconButton label="Quitar producto" icon={<IconClose />} variant="danger" />
          <IconButton label="Deshabilitado" icon={<IconClose />} disabled />
        </Row>
      </Section>

      <Section title="Field, Input y Select">
        <div className={styles.form}>
          <Field label="Nombre del producto" hint="Como aparece en la tirilla">
            <Input placeholder="Cemento gris 50kg" />
          </Field>
          <Field label="Precio de venta" hint="Con IVA incluido">
            <Input inputMode="decimal" align="end" defaultValue="32.900" />
          </Field>
          <Field label="Cantidad" error="La cantidad debe ser mayor que cero">
            <Input inputMode="decimal" align="end" defaultValue="0" />
          </Field>
          <Field label="Código de barras">
            <Input disabled defaultValue="7701234567890" />
          </Field>
          <Field label="Unidad de medida" hint="La unidad base del inventario">
            <Select options={UNITS} placeholder="Elegir unidad" />
          </Field>
          <Field label="Unidad de venta" error="Elija una unidad">
            <Select options={UNITS} placeholder="Elegir unidad" />
          </Field>
          <Field label="Sede">
            <Select options={[{ value: 'principal', label: 'Principal' }]} defaultValue="principal" disabled />
          </Field>
        </div>
      </Section>

      <Section title="Checkbox">
        <div className={styles.stack}>
          <Checkbox label="Permitir stock negativo" hint="Las ventas no se bloquean cuando no hay existencias" />
          <Checkbox label="Producto con IVA" defaultChecked />
          <Checkbox label="Deshabilitado" disabled />
          <Checkbox label="Deshabilitado y marcado" disabled defaultChecked />
        </div>
      </Section>

      <Section title="Table">
        <Table caption="Existencias en la sede Principal" showCaption>
          <TableHead>
            <TableRow>
              <TableHeaderCell>SKU</TableHeaderCell>
              <TableHeaderCell>Producto</TableHeaderCell>
              <TableHeaderCell>Unidad</TableHeaderCell>
              <TableHeaderCell numeric>Cantidad</TableHeaderCell>
              <TableHeaderCell numeric>Precio</TableHeaderCell>
              <TableHeaderCell>Estado</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {STOCK.map((item) => (
              <TableRow
                key={item.sku}
                selected={item.sku === selectedSku}
                onClick={() => setSelectedSku(item.sku)}
              >
                <TableCell>{item.sku}</TableCell>
                <TableCell>{item.name}</TableCell>
                <TableCell>{item.unit}</TableCell>
                <TableCell numeric>{item.qty}</TableCell>
                <TableCell numeric>{item.price}</TableCell>
                <TableCell>
                  {item.state === 'ok' && <Badge tone="success">Disponible</Badge>}
                  {item.state === 'low' && (
                    <Badge tone="warning" icon={<IconAlert />}>
                      Stock bajo
                    </Badge>
                  )}
                  {item.state === 'out' && (
                    <Badge tone="danger" icon={<IconAlert />}>
                      Agotado
                    </Badge>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Section>

      <Section title="Badge">
        <Row>
          <Badge>Borrador</Badge>
          <Badge tone="success">Pagada</Badge>
          <Badge tone="warning">Stock bajo</Badge>
          <Badge tone="danger" icon={<IconAlert />}>
            Rechazada por la DIAN
          </Badge>
          <Badge tone="info">En proceso</Badge>
        </Row>
      </Section>

      <Section title="Dialog">
        <Row>
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
            <div className={styles.form}>
              <Field label="Producto">
                <Input placeholder="Buscar por nombre o código" />
              </Field>
              <Field label="Cantidad">
                <Input inputMode="decimal" align="end" />
              </Field>
              <Field label="Costo unitario">
                <Input inputMode="decimal" align="end" />
              </Field>
            </div>
          </Dialog>
        </Row>
      </Section>

      <Section title="Toast">
        <Row>
          <Button
            variant="secondary"
            onClick={() => toast({ tone: 'success', title: 'Venta guardada', description: 'Factura FE-1024 por $ 128.400' })}
          >
            Éxito
          </Button>
          <Button
            variant="secondary"
            onClick={() =>
              toast({
                tone: 'danger',
                title: 'No se pudo guardar la venta',
                description: 'No hay stock suficiente de Cemento gris 50kg: quedan 3.',
              })
            }
          >
            Error
          </Button>
          <Button
            variant="secondary"
            onClick={() => toast({ tone: 'warning', title: 'Sin conexión', description: 'La venta se enviará al volver la conexión.' })}
          >
            Advertencia
          </Button>
          <Button variant="secondary" onClick={() => toast({ tone: 'info', title: 'Precios actualizados' })}>
            Información
          </Button>
        </Row>
      </Section>

      <Section title="EmptyState">
        <div className={styles.panel}>
          <EmptyState
            icon={<IconBox />}
            headingLevel={3}
            title="Todavía no hay productos"
            description="Cree el primero o impórtelos desde un archivo de Excel."
            action={<Button icon={<IconPlus />}>Crear producto</Button>}
          />
        </div>
      </Section>

      <Section title="Spinner y Skeleton">
        <Row>
          <Spinner size="sm" />
          <Spinner />
          <Spinner size="lg" label="Buscando productos…" />
        </Row>
        <div className={styles.skeletons} aria-busy="true">
          <Skeleton />
          <Skeleton />
          <Skeleton className={styles.short} />
          <div className={styles.skeletonBlock}>
            <Skeleton shape="block" />
          </div>
        </div>
      </Section>
    </main>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section className={styles.section} aria-labelledby={id}>
      <h2 id={id} className={styles.sectionTitle}>
        {title}
      </h2>
      {children}
    </section>
  );
}

function Row({ children }: { children: ReactNode }) {
  return <div className={styles.row}>{children}</div>;
}
