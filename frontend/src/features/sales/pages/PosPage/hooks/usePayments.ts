import { useState } from 'react';
import { isZero } from '@/shared/format';
import { settle } from '../../../model/cart/settlement';
import type { PaymentInput } from '../../../model/paymentMethods';
import {
  amountText,
  nextMethod,
  readRows,
  settlementMessage,
  validPayments,
  type PaymentRow,
} from '../../../model/payments';

// Up to this many payments in one sale.
export const MAX_PAYMENTS = 10;

type Options = {
  total: string;
  /** Sends the sale; the payments are already checked against the total. */
  onConfirm: (payments: PaymentInput[]) => void;
  pending: boolean;
};

/**
 * The payments typed in the checkout and how they settle the total. Every
 * charge starts with one empty cash payment; a new one proposes what is
 * missing. The errors of the amounts show once the cashier submits.
 */
export function usePayments({ total, onConfirm, pending }: Options) {
  const [rows, setRows] = useState<PaymentRow[]>([{ key: 0, method: 'cash', text: '' }]);
  const [submitted, setSubmitted] = useState(false);
  const [problem, setProblem] = useState<string>();
  const read = readRows(rows, total);
  const settlement = settle(total, validPayments(rows, read));

  // Any edit of the payments clears the problem of the last submit.
  function change(key: number, changes: Partial<PaymentRow>) {
    setRows((current) => current.map((r) => (r.key === key ? { ...r, ...changes } : r)));
    setProblem(undefined);
  }

  function add() {
    const method = nextMethod(rows);
    const text = isZero(settlement.missing) ? '' : amountText(settlement.missing);
    setRows((current) => [...current, { key: Math.max(...current.map((r) => r.key)) + 1, method, text }]);
    setProblem(undefined);
  }

  function remove(key: number) {
    setRows((current) => current.filter((r) => r.key !== key));
  }

  function submit() {
    setSubmitted(true);
    if (pending || read.some((r) => r.error)) return;
    const message = settlementMessage(settlement);
    setProblem(message);
    if (!message) onConfirm(validPayments(rows, read));
  }

  const errorOf = (index: number) => (submitted ? read[index]?.error : undefined);
  return { rows, settlement, problem, errorOf, change, add, remove, submit };
}
