import { useState } from 'react';
import { describeError } from '@/shared/api/errors';
import { Checkbox, useToast } from '@/shared/ui';
import { useUpdateStockRules, type StockRules } from '../../../../api/businessRules';
import { SettingsForm } from '../../../../components/SettingsForm/SettingsForm';

/** Whether the stock may go below zero. */
export function StockRulesForm({ rules }: { rules: StockRules }) {
  const [allowNegative, setAllowNegative] = useState(rules.allow_negative_stock);
  const save = useUpdateStockRules();
  const { toast } = useToast();

  return (
    <SettingsForm
      title="Inventario"
      submitLabel="Guardar reglas de inventario"
      saving={save.isPending}
      changed={allowNegative !== rules.allow_negative_stock}
      error={save.isError ? describeError(save.error).message : null}
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate(
          { allow_negative_stock: allowNegative },
          { onSuccess: () => toast({ tone: 'success', title: 'Reglas de inventario guardadas' }) },
        );
      }}
    >
      <Checkbox
        label="Permitir vender sin existencias (stock negativo)"
        hint="Para el negocio que vende antes de registrar la compra. Las existencias quedan en negativo hasta que entre la mercancía."
        checked={allowNegative}
        onChange={(event) => setAllowNegative(event.target.checked)}
      />
    </SettingsForm>
  );
}
