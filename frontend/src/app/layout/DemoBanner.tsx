import { IconInfo } from '@/shared/ui';
import styles from './DemoBanner.module.scss';

// DemoBanner reminds that the company is the public demo: anything can be
// tried, and the night's reset brings its data back. It sits above the main
// content, not in it, and is not a live region: it is read once, not on
// every screen.
export function DemoBanner() {
  return (
    <aside aria-label="Modo demostración" className={styles.root}>
      <IconInfo className={styles.icon} />
      <p className={styles.text}>
        <strong>Modo demostración.</strong> Puede probar todo sin miedo. Los datos se reinician cada noche.
      </p>
    </aside>
  );
}
