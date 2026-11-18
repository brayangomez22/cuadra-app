import clsx from 'clsx';
import { useState, type ComponentProps } from 'react';
import { IconButton } from '../IconButton';
import { IconEye, IconEyeOff } from '../icons';
import { Input } from '../Input';
import styles from './PasswordInput.module.scss';

type Props = Omit<ComponentProps<typeof Input>, 'type' | 'align'>;

/**
 * A password field with a button to see what was typed (OWASP ASVS 2.1.12),
 * inside a Field like Input. Pasting is never blocked (2.1.11): a password
 * manager fills it.
 */
export function PasswordInput({ className, ...props }: Props) {
  const [visible, setVisible] = useState(false);
  return (
    <div className={clsx(styles.root, className)}>
      <Input
        {...props}
        className={styles.input}
        type={visible ? 'text' : 'password'}
        autoCapitalize="off"
        autoCorrect="off"
        spellCheck={false}
      />
      <IconButton
        className={styles.toggle}
        label="Mostrar contraseña"
        aria-pressed={visible}
        icon={visible ? <IconEyeOff /> : <IconEye />}
        onClick={() => {
          setVisible((v) => !v);
        }}
      />
    </div>
  );
}
