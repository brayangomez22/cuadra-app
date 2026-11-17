// errorsShown tells who listens (the usage analytics) that the app showed an
// API error to the user, by its code only: never the message, which may
// name a product or a customer. shared/ knows no feature, so the analytics
// subscribes here instead of being called.

type Listener = (code: string) => void;

const listeners = new Set<Listener>();

/** Listens to the errors shown; returns the function that stops listening. */
export function onErrorShown(listener: Listener) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Reports that an error was shown; without a code it was the network. */
export function errorShown(code: string | undefined) {
  for (const listener of listeners) listener(code ?? 'network_error');
}
