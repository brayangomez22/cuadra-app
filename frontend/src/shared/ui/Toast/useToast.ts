import { createContext, useContext } from 'react';

export type ToastTone = 'success' | 'danger' | 'warning' | 'info';

export type ToastOptions = {
  title: string;
  description?: string;
  tone?: ToastTone;
};

export const ToastContext = createContext<((options: ToastOptions) => void) | null>(null);

/** Shows a brief notification. Needs a ToastProvider above it. */
export function useToast() {
  const toast = useContext(ToastContext);
  if (!toast) throw new Error('useToast must be used inside a ToastProvider');
  return { toast };
}
