import { useState } from 'react';
import { ApiError } from '@/shared/api/errors';
import type { PaymentInput } from '../../../../model/payments/paymentMethods';

/** The sale waiting for an administrator: its payments, and what the customer had left of their limit. */
export type PendingAuthorization = { payments: PaymentInput[]; available: string };

/**
 * A credit sale over the customer's limit: the API rejects it with
 * credit_limit_exceeded and the checkout asks for an administrator's email
 * and password to send it again. Only the payments are kept, never the
 * credentials.
 */
export function useCreditAuthorization() {
  const [pending, setPending] = useState<PendingAuthorization>();

  /** Whether error asks for an authorization; if so, the payments wait for it. */
  function request(error: Error, payments: PaymentInput[]) {
    if (!(error instanceof ApiError) || error.code !== 'credit_limit_exceeded') return false;
    setPending({ payments, available: error.details.available ?? '0' });
    return true;
  }

  return { pending, request, clear: () => setPending(undefined) };
}
