import type { ChangeEvent } from 'react';
import { formatMoney } from '@/shared/format';
import { Checkbox } from '@/shared/ui';

type Props = {
  amount: string;
  /** The location of the user's open drawer. */
  location: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
};

/** Whether the cash comes out of the user's open drawer, recorded as an out movement; on by default where it is offered. */
export function CashOutOption({ amount, location, checked, onChange }: Props) {
  return (
    <Checkbox
      label={`Sacar ${formatMoney(amount)} de mi caja abierta${location ? ` (${location})` : ''}`}
      hint="Queda como egreso de su caja, así el cierre cuadra."
      checked={checked}
      onChange={(event: ChangeEvent<HTMLInputElement>) => onChange(event.target.checked)}
    />
  );
}
