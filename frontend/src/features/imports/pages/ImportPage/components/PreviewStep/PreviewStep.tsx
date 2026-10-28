import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { Alert, Button } from '@/shared/ui';
import { useConfirmImport } from '../../../../api/imports';
import type { ImportPreview } from '../../../../api/types';
import { ImportErrors } from '../../../../components/ImportErrors/ImportErrors';
import { previewSummary } from '../../../../model/previewSummary';
import { SampleTable } from '../SampleTable/SampleTable';
import styles from './PreviewStep.module.scss';

type Props = { preview: ImportPreview; onRestart: () => void };

/**
 * What the API read from the file. A file with errors cannot be confirmed:
 * nothing is imported until it is fixed and uploaded again.
 */
export function PreviewStep({ preview, onRestart }: Props) {
  const navigate = useNavigate();
  const confirm = useConfirmImport();
  const summary = previewSummary(preview.import);
  const valid = preview.import.status === 'validated';

  function start() {
    confirm.mutate(preview.import.id, { onSuccess: (queued) => void navigate(paths.imports.status(queued.id)) });
  }

  return (
    <div className={styles.root}>
      <Alert tone={valid ? 'success' : 'danger'} title={summary.title}>
        {summary.text}
      </Alert>
      {!valid && <ImportErrors importId={preview.import.id} />}
      <SampleTable sample={preview.sample} />
      <div className={styles.footer}>
        <Button variant={valid ? 'ghost' : 'primary'} onClick={onRestart}>
          Subir otro archivo
        </Button>
        {valid && (
          <Button size="lg" loading={confirm.isPending} onClick={start}>
            Importar
          </Button>
        )}
      </div>
    </div>
  );
}
