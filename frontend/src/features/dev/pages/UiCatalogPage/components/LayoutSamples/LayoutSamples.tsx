import { Inline, Stack, type Gap } from '@/shared/ui';
import styles from './LayoutSamples.module.scss';

const gaps: Gap[] = ['1', '2', '4', '6', '8'];

function Box({ label }: { label: string }) {
  return <span className={styles.box}>{label}</span>;
}

/** Stack and Inline with each gap step and the alignments, for the UI catalog. */
export function LayoutSamples() {
  return (
    <Stack gap="6" className={styles.root}>
      <Inline gap="6" align="start" wrap>
        {gaps.map((gap) => (
          <Stack key={gap} gap={gap} align="start">
            <span className={styles.caption}>Stack gap=&quot;{gap}&quot;</span>
            <Box label="Uno" />
            <Box label="Dos" />
          </Stack>
        ))}
      </Inline>
      <Inline gap="3" wrap>
        <Box label="Inline" />
        <Box label='gap="3"' />
        <Box label="wrap" />
      </Inline>
      <Inline justify="between" className={styles.wide}>
        <Box label='justify="between"' />
        <Box label="Acción" />
      </Inline>
    </Stack>
  );
}
