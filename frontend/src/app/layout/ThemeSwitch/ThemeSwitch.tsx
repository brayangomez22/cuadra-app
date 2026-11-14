import { useId } from 'react';
import { IconButton, IconMoon, IconSun } from '@/shared/ui';
import { shownTheme } from './theme';
import { useTheme } from './useTheme';
import styles from './ThemeSwitch.module.scss';

const OPTIONS = [
  { value: 'light', label: 'Tema claro', icon: <IconSun /> },
  { value: 'dark', label: 'Tema oscuro', icon: <IconMoon /> },
] as const;

type Props = {
  /** The folded side menu: one button that switches to the other theme. */
  compact?: boolean;
};

/**
 * ThemeSwitch picks the light or dark theme for this device: a sun and a moon.
 * Until someone picks, the screen follows the OS and the switch marks that one.
 */
export function ThemeSwitch({ compact = false }: Props) {
  const { theme, setTheme } = useTheme();
  const shown = shownTheme(theme);
  // Each switch on the page needs its own radio group.
  const name = useId();

  if (compact) {
    const next = shown === 'dark' ? OPTIONS[0] : OPTIONS[1];
    return (
      <IconButton
        label={`Usar ${next.label.toLowerCase()}`}
        icon={next.icon}
        onClick={() => {
          setTheme(next.value);
        }}
      />
    );
  }

  return (
    <fieldset className={styles.root}>
      <legend className={styles.legend}>Tema</legend>
      <div className={styles.track}>
        {OPTIONS.map((option) => (
          <label key={option.value} className={styles.option} title={option.label}>
            <input
              type="radio"
              name={name}
              className={styles.input}
              aria-label={option.label}
              checked={shown === option.value}
              onChange={() => {
                setTheme(option.value);
              }}
            />
            {option.icon}
          </label>
        ))}
      </div>
    </fieldset>
  );
}
