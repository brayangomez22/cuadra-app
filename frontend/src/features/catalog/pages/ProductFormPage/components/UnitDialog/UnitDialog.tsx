import { useId } from 'react';
import { Alert, Button, Dialog, DialogClose } from '@/shared/ui';
import type { UnitOfMeasure } from '../../../../api/types';
import { hasAxes, variantName } from '../../../../model/variants';
import { useUnitForm, type UnitTarget } from '../../hooks/useUnitForm';
import { UnitFields } from '../UnitFields/UnitFields';
import styles from './UnitDialog.module.scss';

type Props = { target: UnitTarget; units: UnitOfMeasure[]; onClose: () => void };

/** Adds or edits an alternate unit of a variant, with the price proportional to the base one as a suggestion. */
export function UnitDialog({ target, units, onClose }: Props) {
  const formId = useId();
  const { product, variant, unit } = target;
  const unitForm = useUnitForm({ target, units, onSaved: onClose });
  // The base unit is not an alternate one, and a unit the variant has is edited, not added again.
  const taken = new Set(variant.units.map((u) => u.unit.id));
  const options = units
    .filter((u) => u.id !== product.base_unit_id && (u.id === unit?.unit.id || !taken.has(u.id)))
    .map((u) => ({ value: u.id, label: `${u.name} (${u.code})` }));
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={unit ? 'Editar unidad de venta' : 'Agregar unidad de venta'}
      description={variantName(product.name, hasAxes(product) ? variant.label : '')}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={unitForm.form.formState.isSubmitting}>
            Guardar unidad
          </Button>
        </>
      }
    >
      <form
        id={formId}
        className={styles.form}
        noValidate
        // The dialog renders in a portal, but React bubbles its submit to the
        // product's form around this section: it must stop here.
        onSubmit={(event) => {
          event.stopPropagation();
          void unitForm.submit(event);
        }}
      >
        {unitForm.formError && <Alert tone="danger">{unitForm.formError}</Alert>}
        <UnitFields unit={unitForm} options={options} editing={unit !== undefined} />
      </form>
    </Dialog>
  );
}
