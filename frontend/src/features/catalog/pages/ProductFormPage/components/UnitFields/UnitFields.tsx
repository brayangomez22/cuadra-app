import { formatMoney, formatQuantity, toDecimalInput } from '@/shared/format';
import { Button, Field, Input, type SelectOption } from '@/shared/ui';
import { SelectController } from '../../../../components/SelectController/SelectController';
import { useFieldError } from '../../../../hooks/useFieldError';
import type { UnitForm } from '../../hooks/useUnitForm';
import styles from './UnitFields.module.scss';

type Props = {
  unit: UnitForm;
  /** The units it can be: the variant's other ones are edited, not added again. */
  options: SelectOption[];
  /** Editing a unit does not change which one it is. */
  editing: boolean;
};

/** The fields of an alternate unit: which one, how many base units it holds, its price and its barcode. */
export function UnitFields({ unit, options, editing }: Props) {
  const { form, factor, suggested, base, basePrice } = unit;
  const { register, control, setValue } = form;
  const error = useFieldError(control);
  const hint =
    factor && suggested
      ? `Sugerido: ${formatMoney(suggested)} (${formatQuantity(factor)} × ${formatMoney(basePrice)}).`
      : undefined;
  return (
    <>
      <Field label="Unidad de venta" error={error('unitId')}>
        <SelectController
          control={control}
          name="unitId"
          options={options}
          placeholder="Elija la unidad"
          disabled={editing}
        />
      </Field>
      <Field label={`Cuántas ${base} trae cada una`} hint="Puede tener decimales: 0,5 o 2,5." error={error('factor')}>
        <Input inputMode="decimal" align="end" autoComplete="off" {...register('factor')} />
      </Field>
      <Field label="Precio (sin IVA)" hint={hint} error={error('price')}>
        <Input inputMode="decimal" align="end" autoComplete="off" {...register('price')} />
      </Field>
      {suggested && (
        <Button
          variant="ghost"
          className={styles.suggest}
          onClick={() => setValue('price', toDecimalInput(suggested), { shouldValidate: true })}
        >
          Usar el sugerido
        </Button>
      )}
      <Field label="Código de barras" optional error={error('barcode')}>
        <Input autoComplete="off" spellCheck={false} {...register('barcode')} />
      </Field>
    </>
  );
}
