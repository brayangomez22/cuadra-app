import { useFieldError } from '@/shared/forms/useFieldError';
import { Checkbox, Field, Input } from '@/shared/ui';
import type { SalesRules } from '../../../../api/businessRules';
import { SettingsForm } from '../../../../components/SettingsForm/SettingsForm';
import { useSalesRulesForm } from '../../hooks/useSalesRulesForm';

/** What the cashiers may do when they sell. */
export function SalesRulesForm({ rules }: { rules: SalesRules }) {
  const { form, formError, saving, submit } = useSalesRulesForm(rules);
  const error = useFieldError(form.control);

  return (
    <SettingsForm
      title="Ventas"
      submitLabel="Guardar reglas de venta"
      onSubmit={submit}
      saving={saving}
      changed={form.formState.isDirty}
      error={formError}
    >
      <Field
        label="Descuento máximo del cajero (%)"
        hint="Lo más que puede descontar en una línea quien no tiene permiso de dar descuentos. 0 no le deja descontar."
        error={error('maxDiscount')}
      >
        <Input inputMode="decimal" autoComplete="off" {...form.register('maxDiscount')} />
      </Field>
      <Checkbox
        label="Exigir una caja abierta para vender"
        hint="Así cada venta en efectivo queda en una caja que se cuadra al cerrar."
        {...form.register('requireCashSession')}
      />
    </SettingsForm>
  );
}
