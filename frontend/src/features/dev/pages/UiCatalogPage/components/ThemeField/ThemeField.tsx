import { useState } from 'react';
import { Field, Select } from '@/shared/ui';
import { THEMES } from '../../../../model/samples';
import styles from './ThemeField.module.scss';

function applyTheme(theme: string) {
  const root = document.documentElement;
  if (theme === 'system') delete root.dataset.theme;
  else root.dataset.theme = theme;
}

/** Switches the page between the system, light and dark themes. */
export function ThemeField() {
  const [theme, setTheme] = useState('system');
  return (
    <Field label="Tema" className={styles.root}>
      <Select
        options={THEMES}
        value={theme}
        onValueChange={(value) => {
          setTheme(value);
          applyTheme(value);
        }}
      />
    </Field>
  );
}
