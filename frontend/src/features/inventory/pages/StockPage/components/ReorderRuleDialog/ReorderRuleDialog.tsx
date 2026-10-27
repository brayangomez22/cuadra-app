import { useId, useState, type ReactElement } from 'react';
import { Alert, Button, Dialog, DialogClose, Field, Input } from '@/shared/ui';
import type { StockLevel } from '../../../../api/types';
import { useReorderRuleForm } from '../../hooks/useReorderRuleForm';
import styles from './ReorderRuleDialog.module.scss';

type Props = {
  level: StockLevel;
  /** The variant's name, for the description and the toast. */
  name: string;
  /** The button that opens the dialog. */
  trigger: ReactElement;
};

/** Sets or removes the stock at which a variant must be reordered, and up to how much. */
export function ReorderRuleDialog({ level, name, trigger }: Props) {
  const formId = useId();
  const [open, setOpen] = useState(false);
  const rule = useReorderRuleForm({ level, name, onDone: () => setOpen(false) });
  const { errors } = rule.form.formState;

  function changeOpen(next: boolean) {
    if (next) rule.reset();
    setOpen(next);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={changeOpen}
      trigger={trigger}
      title="Mínimo y máximo"
      description={`Cuando la existencia de ${name} llegue al mínimo, aparecerá en «Por reponer» con lo que falta para el máximo.`}
      footer={
        <>
          {level.reorder && (
            <Button variant="danger" loading={rule.isClearing} onClick={rule.clear}>
              Quitar
            </Button>
          )}
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={rule.isSaving}>
            Guardar
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.root} noValidate onSubmit={(event) => void rule.submit(event)}>
        {rule.formError && <Alert tone="danger">{rule.formError}</Alert>}
        <Field label="Stock mínimo" hint="En la unidad del producto." error={errors.min_quantity?.message}>
          <Input align="end" inputMode="decimal" autoComplete="off" {...rule.form.register('min_quantity')} />
        </Field>
        <Field label="Stock máximo" hint="Hasta donde pedir." error={errors.max_quantity?.message}>
          <Input align="end" inputMode="decimal" autoComplete="off" {...rule.form.register('max_quantity')} />
        </Field>
      </form>
    </Dialog>
  );
}
