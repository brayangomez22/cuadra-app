import { useId } from 'react';
import { Button, IconDownload } from '@/shared/ui';
import { useDownloadTemplate } from '../../../../api/template';
import styles from './TemplateDownloads.module.scss';

/** The empty template with the business's columns: its attributes, its units and, with variants, its axes. */
export function TemplateDownloads() {
  const download = useDownloadTemplate();
  const headingId = useId();
  return (
    <section className={styles.root} aria-labelledby={headingId}>
      <h2 id={headingId} className={styles.title}>
        Plantilla
      </h2>
      <p className={styles.text}>
        Tiene las columnas de su negocio. Llénela con una fila por producto (o por variante) y súbala abajo.
      </p>
      <div className={styles.actions}>
        {(['xlsx', 'csv'] as const).map((format) => (
          <Button
            key={format}
            variant="secondary"
            icon={<IconDownload />}
            loading={download.isPending && download.variables === format}
            onClick={() => download.mutate(format)}
          >
            {`Plantilla .${format}`}
          </Button>
        ))}
      </div>
    </section>
  );
}
