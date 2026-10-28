import { Alert, Button, Dialog, DialogClose, Field, Select } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import { useStartCountForm, type CountScope } from '../../hooks/useStartCountForm';
import { CategoryChoice } from './CategoryChoice';
import styles from './StartCountDialog.module.scss';

const scopeOptions: { value: CountScope; label: string }[] = [
  { value: 'all', label: 'Toda la sede' },
  { value: 'categories', label: 'Solo algunas categorías' },
];

const modeOptions = [
  { value: 'sum', label: 'Se suman (cada uno contó una parte)' },
  { value: 'review', label: 'Deben coincidir (si no, se recuenta)' },
];

/** Starts a count at the location: of everything or of some categories, and how two counts of the same product add up. */
export function StartCountDialog({ location }: { location: Location }) {
  const form = useStartCountForm(location.id);
  const formId = `start-count-${location.id}`;
  return (
    <Dialog
      trigger={<Button>Iniciar conteo</Button>}
      title={`Iniciar conteo en ${location.name}`}
      description="Se toma una foto del stock ahora. Pueden seguir vendiendo: lo que se mueva mientras cuentan se descuenta."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={form.isPending}>
            Iniciar conteo
          </Button>
        </>
      }
    >
      <form
        id={formId}
        className={styles.root}
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.submit();
        }}
      >
        {form.error && <Alert tone="danger">{form.error}</Alert>}
        <Field label="Qué contar">
          <Select options={scopeOptions} value={form.scope} onValueChange={(v) => form.setScope(v as CountScope)} />
        </Field>
        {form.scope === 'categories' && <CategoryChoice chosen={form.categoryIds} onToggle={form.toggleCategory} />}
        <Field label="Si dos personas cuentan el mismo producto" hint="Por ejemplo, en dos estantes distintos.">
          <Select options={modeOptions} value={form.mode} onValueChange={(v) => form.setMode(v as 'sum' | 'review')} />
        </Field>
      </form>
    </Dialog>
  );
}
