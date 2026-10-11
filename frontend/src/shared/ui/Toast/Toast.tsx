import * as RadixToast from '@radix-ui/react-toast';
import { useCallback, useState, type ReactNode } from 'react';
import { IconButton } from '../IconButton';
import { IconAlert, IconClose, IconInfo, IconSuccess } from '../icons';
import { ToastContext, type ToastOptions, type ToastTone } from './useToast';
import styles from './Toast.module.scss';

type Item = ToastOptions & { id: number; tone: ToastTone };

const ICONS = { success: IconSuccess, danger: IconAlert, warning: IconAlert, info: IconInfo } as const;

// An error stays until the user closes it (PRODUCT.md: it must be clear when
// something was not saved) and interrupts the screen reader; the rest go away
// on their own and wait their turn.
const DURATION_MS = { success: 5000, info: 5000, warning: 8000, danger: Infinity } as const;

let nextId = 0;

// Radix Toast with a hook to show notifications from anywhere below it. Radix
// pauses them on hover or focus, and F8 moves focus to the region.
export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Item[]>([]);

  const toast = useCallback((options: ToastOptions) => {
    nextId += 1;
    setItems((current) => [...current, { ...options, tone: options.tone ?? 'info', id: nextId }]);
  }, []);

  const remove = (id: number) => setItems((current) => current.filter((item) => item.id !== id));

  return (
    <ToastContext value={toast}>
      <RadixToast.Provider label="Notificación" swipeDirection="right">
        {children}
        {items.map(({ id, tone, title, description }) => {
          const Icon = ICONS[tone];
          return (
            <RadixToast.Root
              key={id}
              className={styles.root}
              data-tone={tone}
              type={tone === 'danger' ? 'foreground' : 'background'}
              duration={DURATION_MS[tone]}
              onOpenChange={(open) => !open && remove(id)}
            >
              <Icon className={styles.icon} />
              <div className={styles.content}>
                <RadixToast.Title className={styles.title}>{title}</RadixToast.Title>
                {description && (
                  <RadixToast.Description className={styles.description}>{description}</RadixToast.Description>
                )}
              </div>
              <RadixToast.Close asChild>
                <IconButton label="Cerrar notificación" icon={<IconClose />} className={styles.close} />
              </RadixToast.Close>
            </RadixToast.Root>
          );
        })}
        <RadixToast.Viewport className={styles.viewport} label="Notificaciones ({hotkey})" />
      </RadixToast.Provider>
    </ToastContext>
  );
}
