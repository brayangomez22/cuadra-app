import { useCategories } from '@/features/catalog';
import { QueryState } from '@/shared/query';
import { Checkbox, Spinner } from '@/shared/ui';
import styles from './StartCountDialog.module.scss';

type Props = { chosen: string[]; onToggle: (id: string, checked: boolean) => void };

/** The categories to count; each one brings its subcategories. */
export function CategoryChoice({ chosen, onToggle }: Props) {
  const categories = useCategories();
  const list = categories.data ?? [];
  return (
    <fieldset className={styles.categories}>
      <legend className={styles.legend}>Categorías (con sus subcategorías)</legend>
      <QueryState
        queries={[categories]}
        errorTitle="No pudimos cargar las categorías"
        loading={<Spinner label="Cargando categorías…" />}
        isEmpty={list.length === 0}
        empty={<p className={styles.empty}>Aún no hay categorías: cuente toda la sede.</p>}
      >
        {() => (
          <div className={styles.options}>
            {list.map((category) => (
              <Checkbox
                key={category.id}
                label={category.name}
                checked={chosen.includes(category.id)}
                onChange={(event) => onToggle(category.id, event.target.checked)}
              />
            ))}
          </div>
        )}
      </QueryState>
    </fieldset>
  );
}
