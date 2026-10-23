import { formatMoney } from '@/shared/format';
import { Badge, Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Product } from '../../../../api/types';
import styles from './VariantList.module.scss';

/** The variants of a product with axes while the module variants is off: shown, not edited. */
export function VariantList({ product }: { product: Product }) {
  return (
    <div className={styles.root}>
      <p className={styles.hint}>Active el módulo de variantes para editarlas.</p>
      <div className={styles.table}>
        <Table caption="Variantes del producto">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Variante</TableHeaderCell>
              <TableHeaderCell>SKU</TableHeaderCell>
              <TableHeaderCell numeric>Precio con IVA</TableHeaderCell>
              <TableHeaderCell>Estado</TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {product.variants.map((variant) => (
              <TableRow key={variant.id}>
                <TableCell className={styles.label}>{variant.label}</TableCell>
                <TableCell className={styles.sku}>{variant.sku}</TableCell>
                <TableCell numeric>{formatMoney(variant.price_with_tax)}</TableCell>
                <TableCell>{variant.active ? 'Activa' : <Badge tone="warning">Inactiva</Badge>}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
