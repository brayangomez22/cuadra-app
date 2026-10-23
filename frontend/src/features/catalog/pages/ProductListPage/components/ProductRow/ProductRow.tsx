import { paths } from '@/app/routes/paths';
import { formatMoney, formatPercent } from '@/shared/format';
import { Badge, TableCell, TableRow, TextLink } from '@/shared/ui';
import type { Product } from '../../../../api/types';
import { variantAmount } from '../../../../model/variantAmount';
import { defaultVariant, hasAxes } from '../../../../model/variants';
import styles from './ProductRow.module.scss';

type Props = { product: Product; canWrite: boolean; canSeeStock: boolean };

const asPercent = (margin: string) => formatPercent(margin, 'percent');

/** A product of the list: its name, price with IVA and, for who may, its cost, margin and links. */
export function ProductRow({ product, canWrite, canSeeStock }: Props) {
  return (
    <TableRow>
      <TableCell>
        <span className={styles.name}>
          {product.name}
          {!product.active && <Badge>Inactivo</Badge>}
        </span>
        {/* SKU (or how many variants) and unit under the name: on a phone the price stays in view. */}
        <span className={styles.meta}>
          {hasAxes(product) ? (
            `${String(product.variants.length)} variantes`
          ) : (
            <span className={styles.sku}>{defaultVariant(product).sku}</span>
          )}{' '}
          · {product.unit.code}
        </span>
      </TableCell>
      <TableCell numeric>
        {variantAmount(product, (v) => v.price_with_tax, formatMoney)}
        <span className={styles.meta}>IVA {formatPercent(product.tax_rate, 'rate')}</span>
      </TableCell>
      {canWrite && <TableCell numeric>{variantAmount(product, (v) => v.cost, formatMoney)}</TableCell>}
      {canWrite && <TableCell numeric>{variantAmount(product, (v) => v.margin_percent, asPercent)}</TableCell>}
      {(canWrite || canSeeStock) && (
        <TableCell>
          <ProductLinks product={product} canWrite={canWrite} canSeeStock={canSeeStock} />
        </TableCell>
      )}
    </TableRow>
  );
}

// The links of the row: its kardex and its form, each for who may open it.
function ProductLinks({ product, canWrite, canSeeStock }: Props) {
  return (
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
  );
}
