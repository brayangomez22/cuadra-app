import { useSearchParams } from 'react-router';
import { useCan } from '@/features/auth';
import { useProducts, variantName, type Product } from '@/features/catalog';
import { formatMoney, formatQuantity, isNegative } from '@/shared/format';
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
  Skeleton,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
  TextLink,
} from '@/shared/ui';
import { PAGE_SIZE, useLocations, useStock, type Location, type StockLevel } from './api';
import { CreateLocationDialog } from './CreateLocationDialog';
import styles from './StockPage.module.scss';

/** /inventario: the stock of one location (sede), chosen in the URL (?sede=). */
export function StockPage() {
  const can = useCan();
  const [params, setParams] = useSearchParams();
  const locations = useLocations();
  const list = locations.data ?? [];
  const location = list.find((l) => l.id === params.get('sede')) ?? list.find((l) => l.active) ?? list[0];
  const pageNumber = Math.max(Number(params.get('pagina')) || 1, 1);
  const offset = (pageNumber - 1) * PAGE_SIZE;

  const chooseLocation = (id: string) => setParams({ sede: id });
  const canManage = can('tenant:manage');
  const canReceive = can('inventory:adjust');

  const actions = (
    <>
      {canReceive && location && (
        <ButtonLink to={`/inventario/entrada?sede=${location.id}`} icon={<IconPlus />}>
          Registrar entrada
        </ButtonLink>
      )}
      {canManage && list.length > 0 && (
        <CreateLocationDialog
          trigger={<Button variant="secondary">Nueva sede</Button>}
          onCreated={(created) => chooseLocation(created.id)}
        />
      )}
    </>
  );

  return (
    <Page title="Inventario" description="Lo que hay en cada sede, en la unidad de cada producto." actions={actions}>
      {locations.isPending ? (
        <Spinner label="Cargando sedes…" className={styles.loading} />
      ) : locations.isError ? (
        <div className={styles.error}>
          <Alert tone="danger" title="No pudimos cargar las sedes">
            {locations.error.message}
          </Alert>
          <Button variant="secondary" onClick={() => void locations.refetch()}>
            Reintentar
          </Button>
        </div>
      ) : !location ? (
        <EmptyState
          icon={<IconBox />}
          title="Aún no hay sedes"
          description={
            canManage
              ? 'Cree la primera sede o bodega para registrar mercancía y ver sus existencias.'
              : 'Pida al dueño que cree la primera sede o bodega en Cuadra.'
          }
          action={
            canManage && (
              <CreateLocationDialog
                trigger={<Button icon={<IconPlus />}>Crear sede</Button>}
                onCreated={(created) => chooseLocation(created.id)}
              />
            )
          }
        />
      ) : (
        <>
          <Field label="Sede" className={styles.location}>
            <Select options={locationOptions(list)} value={location.id} onValueChange={chooseLocation} />
          </Field>
          <LocationStock
            location={location}
            offset={offset}
            canSeeCost={can('catalog:write')}
            canReceive={canReceive}
            onOffsetChange={(next) =>
              setParams({ sede: location.id, ...(next > 0 && { pagina: String(next / PAGE_SIZE + 1) }) })
            }
          />
        </>
      )}
    </Page>
  );
}

function locationOptions(locations: Location[]) {
  return locations.map((l) => ({ value: l.id, label: l.active ? l.name : `${l.name} (inactiva)` }));
}

type StockProps = {
  location: Location;
  offset: number;
  canSeeCost: boolean;
  canReceive: boolean;
  onOffsetChange: (offset: number) => void;
};

function LocationStock({ location, offset, canSeeCost, canReceive, onOffsetChange }: StockProps) {
  const stock = useStock(location.id, offset);
  const levels = stock.data?.items ?? [];
  // The stock has only ids: the names of its products come in one request for
  // the whole page, each product once however many of its variants it shows.
  const productIds = [...new Set(levels.map((level) => level.product_id))];
  const products = useProducts(
    { ids: productIds, status: 'all', limit: PAGE_SIZE },
    { enabled: productIds.length > 0 && !stock.isPlaceholderData },
  );
  const byId = new Map((products.data?.items ?? []).map((product) => [product.id, product]));

  if (stock.isPending) return <Spinner label="Cargando existencias…" className={styles.loading} />;
  if (stock.isError) {
    return (
      <div className={styles.error}>
        <Alert tone="danger" title="No pudimos cargar las existencias">
          {stock.error.message}
        </Alert>
        <Button variant="secondary" onClick={() => void stock.refetch()}>
          Reintentar
        </Button>
      </div>
    );
  }
  if (levels.length === 0) {
    return (
      <EmptyState
        icon={<IconBox />}
        title={`${location.name} aún no tiene existencias`}
        description="Cuando registre una entrada de mercancía, aquí verá cuánto hay de cada producto."
        action={
          canReceive && (
            <ButtonLink to={`/inventario/entrada?sede=${location.id}`} icon={<IconPlus />}>
              Registrar entrada
            </ButtonLink>
          )
        }
      />
    );
  }

  return (
    <div className={styles.results} aria-busy={stock.isPlaceholderData || undefined}>
      {products.isError && (
        <Alert tone="warning">No pudimos cargar los nombres de los productos. Las cantidades sí están al día.</Alert>
      )}
      <Table caption={`Existencias en ${location.name}`}>
        <TableHead>
          <TableRow>
            <TableHeaderCell>Producto</TableHeaderCell>
            <TableHeaderCell numeric>Existencia</TableHeaderCell>
            {canSeeCost && <TableHeaderCell numeric>Costo promedio</TableHeaderCell>}
          </TableRow>
        </TableHead>
        <TableBody>
          {levels.map((level) => (
            <StockRow
              key={level.variant_id}
              level={level}
              product={byId.get(level.product_id)}
              loading={products.isPending}
              canSeeCost={canSeeCost}
            />
          ))}
        </TableBody>
      </Table>
      <Pagination
        label="Páginas de existencias"
        total={stock.data.total}
        limit={PAGE_SIZE}
        offset={offset}
        onOffsetChange={onOffsetChange}
      />
    </div>
  );
}

type RowProps = { level: StockLevel; product?: Product; loading: boolean; canSeeCost: boolean };

function StockRow({ level, product, loading, canSeeCost }: RowProps) {
  const kardex = `/productos/${level.product_id}/kardex?variante=${level.variant_id}&sede=${level.location_id}`;
  const variant = product?.variants.find((v) => v.id === level.variant_id);
  return (
    <TableRow>
      <TableCell>
        {product && variant ? (
          <TextLink tone="quiet" to={kardex}>
            {variantName(product.name, variant.label)}
          </TextLink>
        ) : loading ? (
          <Skeleton className={styles.nameSkeleton} />
        ) : (
          <TextLink tone="quiet" to={kardex}>
            Ver kardex
          </TextLink>
        )}
        {variant && <span className={styles.sku}>{variant.sku}</span>}
      </TableCell>
      <TableCell numeric>
        <span className={styles.quantity}>
          {isNegative(level.quantity) && <Badge tone="danger">Negativa</Badge>}
          {formatQuantity(level.quantity)} {product?.unit.code}
        </span>
      </TableCell>
      {canSeeCost && <TableCell numeric>{level.average_cost ? formatMoney(level.average_cost) : '—'}</TableCell>}
    </TableRow>
  );
}
