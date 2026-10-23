import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router';
import { useCan } from '@/features/auth';
import { formatMoney, formatPercent } from '@/shared/format';
import { useDebouncedValue } from '@/shared/hooks/useDebouncedValue';
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  EmptyState,
  Field,
  IconPlus,
  IconSearch,
  IconTag,
  Input,
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
  TextLink,
} from '@/shared/ui';
import { toUnits } from '@/shared/money/decimal';
import { PAGE_SIZE, useProducts, type Product, type ProductStatus, type ProductVariant } from './api';
import { defaultVariant, hasAxes } from './variants';
import styles from './ProductListPage.module.scss';
import { paths } from '@/app/routes/paths';

// The filter of status as it shows in the URL, which the user sees.
const statusByParam: Record<string, ProductStatus> = { inactivos: 'inactive', todos: 'all' };
const statusOptions = [
  { value: 'active', label: 'Activos' },
  { value: 'inactive', label: 'Inactivos' },
  { value: 'all', label: 'Todos' },
];

export function ProductListPage() {
  const can = useCan();
  const canWrite = can('catalog:write');
  const [params, setParams] = useSearchParams();
  const q = params.get('q') ?? '';
  const status = statusByParam[params.get('estado') ?? ''] ?? 'active';
  const pageNumber = Math.max(Number(params.get('pagina')) || 1, 1);
  const offset = (pageNumber - 1) * PAGE_SIZE;

  // The text is searched once the user stops typing: one request, not one per key.
  const [text, setText] = useState(q);
  const searched = useDebouncedValue(text.trim());
  useEffect(() => {
    if (searched === q) return;
    setParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (searched) next.set('q', searched);
        else next.delete('q');
        next.delete('pagina');
        return next;
      },
      { replace: true },
    );
  }, [searched, q, setParams]);

  function update(changes: Record<string, string | null>) {
    setParams((current) => {
      const next = new URLSearchParams(current);
      for (const [key, value] of Object.entries(changes)) {
        if (value === null) next.delete(key);
        else next.set(key, value);
      }
      return next;
    });
  }

  const products = useProducts({ q: q || undefined, status, limit: PAGE_SIZE, offset });

  return (
    <Page
      title="Productos"
      description="Su catálogo: precios, IVA y unidades."
      actions={
        canWrite && (
          <ButtonLink to={paths.products.new()} icon={<IconPlus />}>
            Nuevo producto
          </ButtonLink>
        )
      }
    >
      <div role="search" className={styles.filters}>
        <Field label="Buscar" className={styles.search}>
          <Input
            type="search"
            value={text}
            onChange={(event) => setText(event.target.value)}
            placeholder="Nombre, SKU o código de barras"
            autoComplete="off"
            enterKeyHint="search"
          />
        </Field>
        <Field label="Estado" className={styles.status}>
          <Select
            options={statusOptions}
            value={status}
            onValueChange={(value) =>
              update({ estado: value === 'inactive' ? 'inactivos' : value === 'all' ? 'todos' : null, pagina: null })
            }
          />
        </Field>
      </div>

      {products.isPending ? (
        <Spinner label="Cargando productos…" className={styles.loading} />
      ) : products.isError ? (
        <div className={styles.error}>
          <Alert tone="danger" title="No pudimos cargar los productos">
            {products.error.message}
          </Alert>
          <Button variant="secondary" onClick={() => void products.refetch()}>
            Reintentar
          </Button>
        </div>
      ) : products.data.items.length === 0 ? (
        <NoProducts q={q} status={status} canWrite={canWrite} />
      ) : (
        <div className={styles.results} aria-busy={products.isPlaceholderData || undefined}>
          <ProductTable products={products.data.items} canWrite={canWrite} canSeeStock={can('inventory:read')} />
          <Pagination
            label="Páginas de productos"
            total={products.data.total}
            limit={PAGE_SIZE}
            offset={offset}
            onOffsetChange={(next) => update({ pagina: next === 0 ? null : String(next / PAGE_SIZE + 1) })}
          />
        </div>
      )}
    </Page>
  );
}

type TableProps = { products: Product[]; canWrite: boolean; canSeeStock: boolean };

function ProductTable({ products, canWrite, canSeeStock }: TableProps) {
  const hasActions = canWrite || canSeeStock;
  return (
    <Table caption="Productos">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Precio con IVA</TableHeaderCell>
          {canWrite && <TableHeaderCell numeric>Costo</TableHeaderCell>}
          {canWrite && <TableHeaderCell numeric>Margen</TableHeaderCell>}
          {hasActions && (
            <TableHeaderCell>
              <span className="srOnly">Acciones</span>
            </TableHeaderCell>
          )}
        </TableRow>
      </TableHead>
      <TableBody>
        {products.map((product) => (
          <TableRow key={product.id}>
            <TableCell>
              <span className={styles.name}>
                {product.name}
                {!product.active && <Badge>Inactivo</Badge>}
              </span>
              {/* SKU (or how many variants) and unit under the name: on a phone the price stays in view. */}
              <span className={styles.meta}>
                {hasAxes(product) ? (
                  `${product.variants.length} variantes`
                ) : (
                  <span className={styles.sku}>{defaultVariant(product).sku}</span>
                )}{' '}
                · {product.unit.code}
              </span>
            </TableCell>
            <TableCell numeric>
              {amount(product, (v) => v.price_with_tax, formatMoney)}
              <span className={styles.meta}>IVA {formatPercent(product.tax_rate, 'rate')}</span>
            </TableCell>
            {canWrite && <TableCell numeric>{amount(product, (v) => v.cost, formatMoney)}</TableCell>}
            {canWrite && (
              <TableCell numeric>
                {amount(
                  product,
                  (v) => v.margin_percent,
                  (m) => formatPercent(m, 'percent'),
                )}
              </TableCell>
            )}
            {hasActions && (
              <TableCell>
                <span className={styles.actions}>
                  {canSeeStock && (
                    <TextLink tone="quiet" className={styles.action} to={paths.inventory.kardex(product.id)}>
                      Kardex <span className="srOnly">de {product.name}</span>
                    </TextLink>
                  )}
                  {canWrite && (
                    <TextLink tone="quiet" className={styles.action} to={paths.products.edit(product.id)}>
                      Editar <span className="srOnly">{product.name}</span>
                    </TextLink>
                  )}
                </span>
              </TableCell>
            )}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function NoProducts({ q, status, canWrite }: { q: string; status: ProductStatus; canWrite: boolean }) {
  if (q) {
    return (
      <EmptyState
        icon={<IconSearch />}
        title={`No encontramos «${q}»`}
        description="Revise cómo lo escribió o busque por el SKU o el código de barras."
      />
    );
  }
  if (status === 'inactive') {
    return <EmptyState icon={<IconTag />} title="No hay productos inactivos" />;
  }
  return (
    <EmptyState
      icon={<IconTag />}
      title="Aún no hay productos"
      description="Cree los productos que vende para usarlos en el inventario y en las ventas."
      action={
        canWrite && (
          <ButtonLink to={paths.products.new()} icon={<IconPlus />}>
            Nuevo producto
          </ButtonLink>
        )
      }
    />
  );
}

/**
 * An amount of the product's variants (the ones for sale, or all if none is):
 * the amount when they share it, "Desde" the smallest when they do not, and
 * "—" when the API does not send it (costs, to those who cannot see them).
 */
function amount(product: Product, of: (v: ProductVariant) => string | undefined, format: (value: string) => string) {
  const forSale = product.variants.filter((v) => v.active);
  const values = (forSale.length > 0 ? forSale : product.variants).map(of);
  const known = values.filter((value): value is string => value !== undefined);
  if (known.length === 0) return '—';
  const smallest = known.reduce((min, value) => (toUnits(value) < toUnits(min) ? value : min));
  return known.every((value) => toUnits(value) === toUnits(smallest)) ? format(smallest) : `Desde ${format(smallest)}`;
}
