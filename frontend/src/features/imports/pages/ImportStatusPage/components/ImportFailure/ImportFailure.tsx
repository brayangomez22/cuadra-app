import { paths } from '@/app/routes/paths';
import { Alert, ButtonLink } from '@/shared/ui';
import { ImportErrors } from '../../../../components/ImportErrors/ImportErrors';
import styles from './ImportFailure.module.scss';

/**
 * An import that failed, with its errors, or one that was never confirmed
 * (no importId). The batches written before the failure stay: uploading the
 * file again with "update" completes what is missing.
 */
export function ImportFailure({ importId }: { importId?: string }) {
  return (
    <section className={styles.root}>
      {importId ? (
        <Alert tone="danger" title="La importación no terminó">
          Lo importado antes del error quedó guardado. Corrija el archivo y súbalo otra vez con «Actualizar el
          producto»: así se completa lo que falta sin duplicar nada.
        </Alert>
      ) : (
        <Alert tone="info" title="Esta importación no se confirmó">
          No se importó nada. Suba el archivo otra vez para revisarlo y confirmarlo.
        </Alert>
      )}
      {importId && <ImportErrors importId={importId} />}
      <div className={styles.actions}>
        <ButtonLink to={paths.imports.new()}>Subir el archivo otra vez</ButtonLink>
      </div>
    </section>
  );
}
