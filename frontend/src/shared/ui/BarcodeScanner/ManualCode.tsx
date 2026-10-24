import { useState, type SyntheticEvent } from 'react';
import { Button } from '../Button';
import { Field } from '../Field';
import { Input } from '../Input';
import styles from './BarcodeScanner.module.scss';

/** The way out when the camera cannot read: a damaged label, poor light, no permission. */
export function ManualCode({ onCode }: { onCode: (code: string) => void }) {
  const [code, setCode] = useState('');

  function submit(event: SyntheticEvent) {
    event.preventDefault();
    const typed = code.trim();
    if (!typed) return;
    onCode(typed);
    setCode('');
  }

  return (
    <form className={styles.manual} onSubmit={submit}>
      <Field label="Digitar el código" className={styles.manualField}>
        <Input
          value={code}
          onChange={(event) => setCode(event.target.value)}
          inputMode="text"
          autoComplete="off"
          spellCheck={false}
          enterKeyHint="done"
        />
      </Field>
      <Button type="submit" variant="secondary">
        Usar el código
      </Button>
    </form>
  );
}
