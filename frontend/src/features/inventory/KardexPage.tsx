import { useParams, useSearchParams } from 'react-router';
import { useCan } from '@/features/auth';
import { unitShort, useProduct } from '@/features/catalog';
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

const ALL_LOCATIONS = 'all';

/** /productos/:productId/kardex: the movements of a product, newest first, optionally of one location (?sede=). */
export function KardexPage() {
  const { productId = '' } = useParams();
  const can = useCan();
  const [params, setParams] = useSearchParams();
  const locationId = params.get('sede') ?? undefined;
  const pageNumber = Math.max(Number(params.get('pagina')) || 1, 1);
  const offset = (pageNumber - 1) * PAGE_SIZE;

  const product = useProduct(productId);
  const locations = useLocations();
  const kardex = useKardex(productId, locationId, offset);

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

  const unit = unitShort(product.data.base_unit);
  const canSeeCost = can('catalog:write');
  const locationName = new Map((locations.data ?? []).map((l) => [l.id, l.name]));
  const receiptLink = `/inventario/entrada?producto=${productId}${locationId ? `&sede=${locationId}` : ''}`;
  const locationOptions = [
    { value: ALL_LOCATIONS, label: 'Todas las sedes' },
    ...(locations.data ?? []).map((l) => ({ value: l.id, label: l.name })),
  ];

  return (
    <Page
      title={product.data.name}
      description={
        <span className={styles.subtitle}>
          Kardex · {product.data.sku} · en {unit}
          {!product.data.active && <Badge tone="warning">Inactivo</Badge>}
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
      <Field label="Sede" className={styles.location}>
        <Select
          options={locationOptions}
          value={locationId ?? ALL_LOCATIONS}
          onValueChange={(value) => setParams(value === ALL_LOCATIONS ? {} : { sede: value })}
          disabled={locations.isPending}
        />
      </Field>

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
          title="Este producto aún no tiene movimientos"
          description={
            locationId
              ? 'No hay entradas, ventas ni ajustes en esta sede.'
              : 'Las entradas, ventas, ajustes y traslados aparecerán aquí.'
          }
        />
      ) : (
        <div className={styles.results} aria-busy={kardex.isPlaceholderData || undefined}>
          <Table caption={`Movimientos de ${product.data.name}`}>
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
            onOffsetChange={(next) =>
              setParams({
                ...(locationId && { sede: locationId }),
                ...(next > 0 && { pagina: String(next / PAGE_SIZE + 1) }),
              })
            }
          />
        </div>
      )}
    </Page>
  );
}
