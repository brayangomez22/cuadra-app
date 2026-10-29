import { IconInfo } from '@/shared/ui';
import styles from './DemoBanner.module.scss';

// DemoBanner reminds that the company is the public demo: anything can be
// tried, and the night's reset brings its data back.
export function DemoBanner() {
  return (
    <p role="status" aria-label="Modo demostración" className={styles.root}>
      <IconInfo className={styles.icon} />
      <span>
        <strong>Modo demostración.</strong> Puede probar todo sin miedo. Los datos se reinician cada noche.
      </span>
    </p>
  );
}
