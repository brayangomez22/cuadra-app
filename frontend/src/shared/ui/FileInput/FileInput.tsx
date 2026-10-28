import clsx from 'clsx';
import type { ComponentProps } from 'react';
import { useFieldControl } from '../Field';
import styles from './FileInput.module.scss';

type Props = Omit<ComponentProps<'input'>, 'type' | 'value' | 'onChange'> & {
  /** The file chosen, or null when the choice is cleared. */
  onFile: (file: File | null) => void;
};

/**
 * FileInput is the native file picker, styled like the other controls: the
 * browser shows the name of the file chosen and opens the camera or the
 * files on a phone. One file at a time.
 */
export function FileInput({ onFile, className, ...props }: Props) {
  const rest = useFieldControl(props);
  return (
    <input
      type="file"
      className={clsx(styles.root, className)}
      onChange={(event) => {
        onFile(event.target.files?.[0] ?? null);
      }}
      {...rest}
    />
  );
}
