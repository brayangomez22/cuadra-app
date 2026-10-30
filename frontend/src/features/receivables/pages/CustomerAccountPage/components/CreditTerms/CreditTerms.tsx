import { useEffect, useRef, useState } from 'react';
import { Button, IconEdit } from '@/shared/ui';
import type { CustomerCredit } from '../../../../api/types';
import { CreditTermsForm } from '../CreditTermsForm/CreditTermsForm';

type Props = { credit: CustomerCredit };

/**
 * The credit limit and term stay out of the way until someone asks to change
 * them: the account is read far more often than its limit is changed.
 */
export function CreditTerms({ credit }: Props) {
  const [editing, setEditing] = useState(false);
  const opener = useRef<HTMLButtonElement>(null);
  const closed = useRef(false);
  // After the form closes, the focus goes back to the button that opened it.
  useEffect(() => {
    if (!editing && closed.current) opener.current?.focus();
  }, [editing]);

  function close() {
    closed.current = true;
    setEditing(false);
  }

  if (editing) return <CreditTermsForm credit={credit} onDone={close} />;
  return (
    <Button ref={opener} variant="secondary" icon={<IconEdit />} onClick={() => setEditing(true)}>
      Cambiar cupo y plazo
    </Button>
  );
}
