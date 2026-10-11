import * as RadixDialog from '@radix-ui/react-dialog';
import clsx from 'clsx';
import type { ReactElement, ReactNode } from 'react';
import { IconButton } from '../IconButton';
import { IconClose } from '../icons';
import styles from './Dialog.module.scss';

type Props = {
  /** Controlled mode. Without it, the trigger opens and closes the dialog. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** One element, usually a Button, that opens the dialog. */
  trigger?: ReactElement;
  /** Accessible name of the dialog. */
  title: ReactNode;
  /** Accessible description: what happens and why. */
  description: ReactNode;
  children?: ReactNode;
  /** Actions, the main one last. Wrap the cancel button in DialogClose. */
  footer?: ReactNode;
  size?: 'sm' | 'md';
  className?: string;
};

// Radix Dialog with our styles: it traps focus, closes with Escape and returns
// focus to the trigger (decision in docs/decisiones.md).
export function Dialog({
  open,
  onOpenChange,
  trigger,
  title,
  description,
  children,
  footer,
  size = 'sm',
  className,
}: Props) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      {trigger && <RadixDialog.Trigger asChild>{trigger}</RadixDialog.Trigger>}
      <RadixDialog.Portal>
        <RadixDialog.Overlay className={styles.overlay} />
        <RadixDialog.Content className={clsx(styles.root, className)} data-size={size}>
          <header className={styles.header}>
            <div className={styles.heading}>
              <RadixDialog.Title className={styles.title}>{title}</RadixDialog.Title>
              <RadixDialog.Description className={styles.description}>{description}</RadixDialog.Description>
            </div>
            <RadixDialog.Close asChild>
              <IconButton label="Cerrar" icon={<IconClose />} />
            </RadixDialog.Close>
          </header>
          {children && <div className={styles.body}>{children}</div>}
          {footer && <footer className={styles.footer}>{footer}</footer>}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}

/** Closes the dialog it is in. Use it with asChild around a Button. */
export const DialogClose = RadixDialog.Close;
