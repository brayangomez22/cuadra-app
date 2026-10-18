import { IconButton, IconEdit, IconTrash } from '@/shared/ui';
import styles from './RowActions.module.scss';

type Props = {
  /** What the row is, for the buttons' names: "Editar Metro", "Borrar Metro". */
  name: string;
  onEdit: () => void;
  onDelete: () => void;
};

/** Edit and delete at the end of a settings row. */
export function RowActions({ name, onEdit, onDelete }: Props) {
  return (
    <span className={styles.root}>
      <IconButton label={`Editar ${name}`} icon={<IconEdit />} onClick={onEdit} />
      <IconButton label={`Borrar ${name}`} icon={<IconTrash />} variant="danger" onClick={onDelete} />
    </span>
  );
}
