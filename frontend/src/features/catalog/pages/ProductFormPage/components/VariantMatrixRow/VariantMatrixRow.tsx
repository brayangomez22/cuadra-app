import { useFormContext, useWatch } from 'react-hook-form';
import { Badge, Checkbox, Field, Input, TableCell, TableRow } from '@/shared/ui';
import { useFieldError } from '@/shared/forms/useFieldError';
import type { ProductFormInput, VariantRow } from '../../../../model/productSchema';
import { axisLabel, generatedSku } from '../../../../model/variantMatrix';
import styles from './VariantMatrixRow.module.scss';

type Props = { index: number; row: VariantRow; inactive: boolean };

const code = { autoComplete: 'off', spellCheck: false } as const;
const amount = { className: styles.amount, inputMode: 'decimal', align: 'end', autoComplete: 'off' } as const;

/** One variant of the matrix. Its empty fields show, as placeholders, what they take from the product. */
export function VariantMatrixRow({ index, row, inactive }: Props) {
  const { register, control } = useFormContext<ProductFormInput>();
  const error = useFieldError(control, `variants.${index}`);
  const [base, baseCost, basePrice] = useWatch({ control, name: ['sku', 'cost', 'price'] });
  const label = axisLabel(row.values);
  // The label of each cell is for screen readers: the column says it.
  const cell = (field: 'sku' | 'barcode' | 'cost' | 'price', text: string) => ({
    label: <span className="srOnly">{`${text} de ${label}`}</span>,
    error: error(`variants.${index}.${field}`),
    className: styles.cell,
  });
  return (
    <TableRow>
      <TableCell className={styles.label}>{label}</TableCell>
      <TableCell>
        <Field {...cell('sku', 'SKU')}>
          <Input
            {...code}
            className={styles.sku}
            autoCapitalize="characters"
            placeholder={generatedSku(base, row.values)}
            {...register(`variants.${index}.sku`)}
          />
        </Field>
      </TableCell>
      <TableCell>
        <Field {...cell('barcode', 'Código de barras')}>
          <Input {...code} {...register(`variants.${index}.barcode`)} />
        </Field>
      </TableCell>
      <TableCell numeric>
        <Field {...cell('cost', 'Costo')}>
          <Input {...amount} placeholder={baseCost} {...register(`variants.${index}.cost`)} />
        </Field>
      </TableCell>
      <TableCell numeric>
        <Field {...cell('price', 'Precio')}>
          <Input
            {...amount}
            placeholder={basePrice}
            disabled={inactive && Boolean(row.variantId)}
            {...register(`variants.${index}.price`)}
          />
        </Field>
      </TableCell>
      <TableCell>
        <RowStatus index={index} row={row} label={label} />
      </TableCell>
    </TableRow>
  );
}

// A saved variant can be deactivated; a new one says so until it is saved.
function RowStatus({ index, row, label }: { index: number; row: VariantRow; label: string }) {
  const { register } = useFormContext<ProductFormInput>();
  if (!row.variantId) return <Badge>Nueva</Badge>;
  return (
    <Checkbox label={<span className="srOnly">{`Activa: ${label}`}</span>} {...register(`variants.${index}.active`)} />
  );
}
