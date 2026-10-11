import * as RadixDialog from '@radix-ui/react-dialog';
import clsx from 'clsx';
import type { ReactElement, ReactNode } from 'react';
import { IconButton } from '../IconButton';
import { IconClose } from '../icons';
import styles from './Drawer.module.scss';

type Props = {
  /** Controlled mode. Without it, the trigger opens and closes the drawer. */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** One element, usually an IconButton, that opens the drawer. */
  trigger?: ReactElement;
  /** Accessible name of the panel, shown in its header ("Menú"). */
  title: string;
  children?: ReactNode;
  className?: string;
};

// Drawer is a side panel over the page, such as the app menu on a phone. It is a
// Radix Dialog: it traps focus, closes with Escape or a tap outside, locks the
// page scroll and returns focus to the trigger.
export function Drawer({ open, onOpenChange, trigger, title, children, className }: Props) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      {trigger && <RadixDialog.Trigger asChild>{trigger}</RadixDialog.Trigger>}
      <RadixDialog.Portal>
        <RadixDialog.Overlay className={styles.overlay} />
        {/* No description: the title and the content speak for themselves. */}
        <RadixDialog.Content className={clsx(styles.root, className)} aria-describedby={undefined}>
          <header className={styles.header}>
            <RadixDialog.Title className={styles.title}>{title}</RadixDialog.Title>
            <RadixDialog.Close asChild>
              <IconButton label={`Cerrar ${title.toLowerCase()}`} icon={<IconClose />} />
            </RadixDialog.Close>
          </header>
          <div className={styles.body}>{children}</div>
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  );
}

/** Closes the drawer it is in. Use it with asChild around a link or button. */
export const DrawerClose = RadixDialog.Close;
