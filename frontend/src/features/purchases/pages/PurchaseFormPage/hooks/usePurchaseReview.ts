import { useRef, useState } from 'react';
import type { PurchaseValues } from '../../../model/purchaseForm/purchaseSchema';
import type { PurchaseForm } from './useSavePurchase';

type Save = { save: (values: PurchaseValues) => Promise<void>; showError: (error: unknown) => void };

/**
 * The step between the form and the API: a valid form opens the review, and
 * only its confirmation registers. A failed registration closes the review
 * and, once its dialog gave the focus back, shows the error on its field.
 */
export function usePurchaseReview(form: PurchaseForm, { save, showError }: Save) {
  const [review, setReview] = useState<PurchaseValues | null>(null);
  const [saving, setSaving] = useState(false);
  const failed = useRef<{ error: unknown } | null>(null);

  const open = form.handleSubmit((values) => {
    setReview(values);
  });

  async function confirm() {
    if (!review) return;
    setSaving(true);
    try {
      await save(review);
    } catch (error) {
      failed.current = { error };
      setReview(null);
    } finally {
      setSaving(false);
    }
  }

  function onCloseAutoFocus(event: Event) {
    const pending = failed.current;
    if (!pending) return;
    event.preventDefault();
    failed.current = null;
    showError(pending.error);
  }

  return { review, saving, open, confirm, cancel: () => setReview(null), onCloseAutoFocus };
}
