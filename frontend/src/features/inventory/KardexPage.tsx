import { useParams, useSearchParams } from 'react-router';
import { useCan } from '@/features/auth';
import { defaultVariant, hasAxes, useProduct, variantName } from '@/features/catalog';
import { formatDateTime, formatMoney, formatQuantity } from '@/shared/format';
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  EmptyState,
  Field,
  IconBox,
  IconPlus,
  Page,
  Pagination,
  Select,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from '@/shared/ui';
import { PAGE_SIZE, useKardex, useLocations } from './api';
import { movementLabels } from './movements';
import styles from './KardexPage.module.scss';
import { paths } from '@/app/routes/paths';

const ALL_LOCATIONS = 'all';

/**
 * /productos/:productId/kardex: the movements of a variant of the product
 * (?variante=, by default its default variant), newest first, optionally of
 * one location (?sede=).
 */
export function KardexPage() {
  const { productId = '' } = useParams();
  const can = useCan();
  const [params, setParams] = useSearchParams();
  const variantParam = params.get('variante') ?? undefined;
  const locationId = params.get('sede') ?? undefined;
  const pageNumber = Math.max(Number(params.get('pagina')) || 1, 1);
  const offset = (pageNumber - 1) * PAGE_SIZE;

  const product = useProduct(productId);
  const locations = useLocations();
  const variant = product.data
    ? (product.data.variants.find((v) => v.id === variantParam) ?? defaultVariant(product.data))
    : undefined;
  const kardex = useKardex(variant?.id, locationId, offset);

  // Changing a filter keeps the others and goes back to the first page.
  function show(next: { variante?: string; sede?: string; pagina?: number }) {
    const variante = 'variante' in next ? next.variante : variantParam;
    const sede = 'sede' in next ? next.sede : locationId;
    setParams({
      ...(variante && { variante }),
      ...(sede && { sede }),
      ...(next.pagina && next.pagina > 1 && { pagina: String(next.pagina) }),
    });
  }

  if (product.isPending) {
    return (
      <Page title="Kardex">
        <Spinner label="Cargando el producto…" className={styles.loading} />
      </Page>
    );
  }
  if (product.isError) {
    return (
      <Page title="Kardex">
        <Alert tone="danger" title="No pudimos abrir el producto">
          {product.error.message}
        </Alert>
      </Page>
    );
  }

  const shown = variant ?? defaultVariant(product.data);
  const withAxes = hasAxes(product.data);
  const name = variantName(product.data.name, shown.label);
  const canSeeCost = can('catalog:write');
  const locationName = new Map((locations.data ?? []).map((l) => [l.id, l.name]));
  const receiptLink = paths.inventory.receipt({ producto: productId, variante: shown.id, sede: locationId });
  const locationOptions = [
    { value: ALL_LOCATIONS, label: 'Todas las sedes' },
    ...(locations.data ?? []).map((l) => ({ value: l.id, label: l.name })),
  ];

  return (
    <Page
      title={name}
      description={
        <span className={styles.subtitle}>
          Kardex · {shown.sku} · en {product.data.unit.code}
          {!product.data.active && <Badge tone="warning">Inactivo</Badge>}
          {product.data.active && !shown.active && <Badge tone="warning">Variante inactiva</Badge>}
        </span>
      }
      actions={
        can('inventory:adjust') && (
          <ButtonLink to={receiptLink} icon={<IconPlus />}>
            Registrar entrada
          </ButtonLink>
        )
      }
    >
      <div className={styles.filters}>
        {withAxes && (
          <Field label="Variante" className={styles.filter}>
            <Select
              options={product.data.variants.map((v) => ({ value: v.id, label: v.label || v.sku }))}
              value={shown.id}
              onValueChange={(value) => show({ variante: value })}
            />
          </Field>
        )}
        <Field label="Sede" className={styles.filter}>
          <Select
            options={locationOptions}
            value={locationId ?? ALL_LOCATIONS}
            onValueChange={(value) => show({ sede: value === ALL_LOCATIONS ? undefined : value })}
            disabled={locations.isPending}
          />
        </Field>
      </div>

      {kardex.isPending ? (
        <Spinner label="Cargando movimientos…" className={styles.loading} />
      ) : kardex.isError ? (
        <div className={styles.error}>
          <Alert tone="danger" title="No pudimos cargar los movimientos">
            {kardex.error.message}
          </Alert>
          <Button variant="secondary" onClick={() => void kardex.refetch()}>
            Reintentar
          </Button>
        </div>
      ) : kardex.data.items.length === 0 ? (
        <EmptyState
          icon={<IconBox />}
          title={withAxes ? 'Esta variante aún no tiene movimientos' : 'Este producto aún no tiene movimientos'}
          description={
            locationId
              ? 'No hay entradas, ventas ni ajustes en esta sede.'
              : 'Las entradas, ventas, ajustes y traslados aparecerán aquí.'
          }
        />
      ) : (
        <div className={styles.results} aria-busy={kardex.isPlaceholderData || undefined}>
          <Table caption={`Movimientos de ${name}`}>
            <TableHead>
              <TableRow>
                <TableHeaderCell>Fecha</TableHeaderCell>
                <TableHeaderCell>Movimiento</TableHeaderCell>
                <TableHeaderCell numeric>Cantidad</TableHeaderCell>
                <TableHeaderCell numeric>Saldo en la sede</TableHeaderCell>
                {canSeeCost && <TableHeaderCell numeric>Costo unitario</TableHeaderCell>}
                <TableHeaderCell>Detalle</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {kardex.data.items.map((movement) => (
                <TableRow key={movement.id}>
                  <TableCell className={styles.date}>{formatDateTime(movement.occurred_at)}</TableCell>
                  <TableCell>
                    {movementLabels[movement.type]}
                    <span className={styles.meta}>{locationName.get(movement.location_id)}</span>
                  </TableCell>
                  <TableCell numeric>
                    <span data-sign={movement.quantity.startsWith('-') ? 'out' : 'in'} className={styles.quantity}>
                      {formatQuantity(movement.quantity, { signed: true })}
                    </span>
                  </TableCell>
                  <TableCell numeric>{formatQuantity(movement.balance_after)}</TableCell>
                  {canSeeCost && (
                    <TableCell numeric>{movement.unit_cost ? formatMoney(movement.unit_cost) : '—'}</TableCell>
                  )}
                  <TableCell className={styles.detail}>{movement.reason ?? ''}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Pagination
            label="Páginas del kardex"
            total={kardex.data.total}
            limit={PAGE_SIZE}
            offset={offset}
            onOffsetChange={(next) => show({ pagina: next / PAGE_SIZE + 1 })}
          />
        </div>
      )}
    </Page>
  );
}
