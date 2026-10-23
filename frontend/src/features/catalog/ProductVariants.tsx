import { useState } from 'react';
import { useFieldArray, useWatch, type Control, type FieldErrors, type UseFormRegister } from 'react-hook-form';
import { formatMoney } from '@/shared/format';
import {
  Alert,
  Badge,
  Checkbox,
  Field,
  Input,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from '@/shared/ui';
import type { Product, VariantAxis } from './api';
import type { ProductFormInput, VariantRow } from './schemas';
import { axisLabel, combinations, comboKey, generatedSku } from './variantMatrix';
import styles from './ProductVariants.module.scss';

// The API's limits: up to 5 axes per product. The matrix stops at 100
// combinations, a sane number of variants to create at once.
const MAX_AXES = 5;
const MAX_COMBINATIONS = 100;

type EditorProps = {
  axes: VariantAxis[];
  control: Control<ProductFormInput>;
  register: UseFormRegister<ProductFormInput>;
  errors: FieldErrors<ProductFormInput>['variants'];
  /** An inactive product's prices cannot change. */
  inactive: boolean;
};

/**
 * The variants of a product (module variants): the person picks values of
 * the tenant's axes and each combination is a variant with its own SKU,
 * barcode, cost and price. Empty fields take the product's. With no value
 * picked, the product is simple.
 */
export function VariantEditor({ axes, control, register, errors, inactive }: EditorProps) {
  const { fields, replace } = useFieldArray({ control, name: 'variants' });
  const [base, baseCost, basePrice] = useWatch({ control, name: ['sku', 'cost', 'price'] });
  // The rows as typed: the fields only hold what each row started with.
  const rows = useWatch({ control, name: 'variants' });
  const [chosen, setChosen] = useState(() => chosenOf(fields));
  const [tooMany, setTooMany] = useState(false);

  // A variant that exists keeps its axes and values: variants are not deleted
  // and every variant of a product names the same axes.
  const existing = rows.filter((row) => row.variantId);
  const lockedAxes = existing.length > 0 ? Object.keys(existing[0]?.values ?? {}) : undefined;
  const allAxes = withAxesOf(axes, rows);
  const chosenAxes = Object.keys(chosen).filter((axis) => (chosen[axis] ?? []).length > 0);

  function toggle(axis: string, value: string) {
    const current = chosen[axis] ?? [];
    const next = {
      ...chosen,
      [axis]: current.includes(value) ? current.filter((v) => v !== value) : [...current, value],
    };
    const combos = combinations(allAxes, next);
    if (combos.length > MAX_COMBINATIONS) {
      setTooMany(true);
      return;
    }
    setTooMany(false);
    setChosen(next);
    const byKey = new Map(rows.map((row) => [comboKey(row.values), row]));
    replace(
      combos.map(
        (values): VariantRow =>
          byKey.get(comboKey(values)) ?? { values, sku: '', barcode: '', cost: '', price: '', active: true },
      ),
    );
  }

  return (
    <div className={styles.root}>
      <div className={styles.axes}>
        {allAxes.map((axis) => {
          const axisLocked =
            (lockedAxes !== undefined && !lockedAxes.includes(axis.name)) ||
            (!chosenAxes.includes(axis.name) && chosenAxes.length >= MAX_AXES);
          const usedValues = new Set(existing.map((row) => row.values[axis.name]));
          return (
            <fieldset key={axis.name} className={styles.axis}>
              <legend className={styles.axisName}>{capitalize(axis.name)}</legend>
              <div className={styles.values}>
                {axis.values.map((value) => (
                  <Checkbox
                    key={value}
                    label={value}
                    checked={(chosen[axis.name] ?? []).includes(value)}
                    disabled={axisLocked || usedValues.has(value)}
                    onChange={() => toggle(axis.name, value)}
                  />
                ))}
              </div>
            </fieldset>
          );
        })}
      </div>
      {lockedAxes && lockedAxes.length > 0 && (
        <p className={styles.hint}>
          Sus variantes varían en {lockedAxes.join(' y ')}: puede agregar valores, pero no quitar los que ya tienen
          variantes.
        </p>
      )}
      {tooMany && (
        <Alert tone="warning">
          Son más de {MAX_COMBINATIONS} variantes. Elija menos valores y agregue el resto después.
        </Alert>
      )}

      {fields.length > 0 && (
        <div className={styles.matrix}>
          <Table caption="Variantes del producto">
            <TableHead>
              <TableRow>
                <TableHeaderCell>Variante</TableHeaderCell>
                <TableHeaderCell>SKU</TableHeaderCell>
                <TableHeaderCell>Código de barras</TableHeaderCell>
                <TableHeaderCell numeric>Costo</TableHeaderCell>
                <TableHeaderCell numeric>Precio (sin IVA)</TableHeaderCell>
                <TableHeaderCell>Activa</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {fields.map((field, index) => {
                // The row as it is now: a variant saved in an earlier attempt has its id.
                const row = rows[index] ?? field;
                const label = axisLabel(row.values);
                const rowErrors = errors?.[index];
                const hidden = (text: string) => <span className="srOnly">{`${text} de ${label}`}</span>;
                return (
                  <TableRow key={field.id}>
                    <TableCell className={styles.label}>{label}</TableCell>
                    <TableCell>
                      <Field label={hidden('SKU')} error={rowErrors?.sku?.message} className={styles.cell}>
                        <Input
                          className={styles.sku}
                          autoComplete="off"
                          autoCapitalize="characters"
                          spellCheck={false}
                          placeholder={generatedSku(base, row.values)}
                          {...register(`variants.${index}.sku`)}
                        />
                      </Field>
                    </TableCell>
                    <TableCell>
                      <Field
                        label={hidden('Código de barras')}
                        error={rowErrors?.barcode?.message}
                        className={styles.cell}
                      >
                        <Input autoComplete="off" spellCheck={false} {...register(`variants.${index}.barcode`)} />
                      </Field>
                    </TableCell>
                    <TableCell numeric>
                      <Field label={hidden('Costo')} error={rowErrors?.cost?.message} className={styles.cell}>
                        <Input
                          className={styles.amount}
                          inputMode="decimal"
                          align="end"
                          autoComplete="off"
                          placeholder={baseCost}
                          {...register(`variants.${index}.cost`)}
                        />
                      </Field>
                    </TableCell>
                    <TableCell numeric>
                      <Field label={hidden('Precio')} error={rowErrors?.price?.message} className={styles.cell}>
                        <Input
                          className={styles.amount}
                          inputMode="decimal"
                          align="end"
                          autoComplete="off"
                          placeholder={basePrice}
                          disabled={inactive && Boolean(row.variantId)}
                          {...register(`variants.${index}.price`)}
                        />
                      </Field>
                    </TableCell>
                    <TableCell>
                      {row.variantId ? (
                        <Checkbox
                          label={<span className="srOnly">{`Activa: ${label}`}</span>}
                          {...register(`variants.${index}.active`)}
                        />
                      ) : (
                        <Badge>Nueva</Badge>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

/** The variants of a product with axes while the module variants is off: shown, not edited. */
export function VariantList({ product }: { product: Product }) {
  return (
    <div className={styles.root}>
      <p className={styles.hint}>Active el módulo de variantes para editarlas.</p>
      <div className={styles.matrix}>
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

// chosenOf is the values the rows use, by axis.
function chosenOf(rows: VariantRow[]): Record<string, string[]> {
  const chosen: Record<string, string[]> = {};
  for (const row of rows) {
    for (const [axis, value] of Object.entries(row.values)) {
      const values = (chosen[axis] ??= []);
      if (!values.includes(value)) values.push(value);
    }
  }
  return chosen;
}

// withAxesOf adds to the tenant's axes the axes and values its variants use
// that the tenant no longer defines, so they show and keep their variants.
function withAxesOf(axes: VariantAxis[], rows: VariantRow[]): VariantAxis[] {
  const chosen = chosenOf(rows);
  const known = axes.map((axis) => ({
    ...axis,
    values: [...axis.values, ...(chosen[axis.name] ?? []).filter((value) => !axis.values.includes(value))],
  }));
  const unknown = Object.keys(chosen)
    .filter((name) => !axes.some((axis) => axis.name === name))
    .map((name) => ({ id: name, name, values: chosen[name] ?? [] }));
  return [...known, ...unknown];
}

function capitalize(text: string) {
  return text.charAt(0).toUpperCase() + text.slice(1);
}
