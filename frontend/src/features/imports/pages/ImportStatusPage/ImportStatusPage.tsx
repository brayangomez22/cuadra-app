import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Page } from '@/shared/ui';
import { useImport } from '../../api/imports';
import type { Import } from '../../api/types';
import { importStage } from '../../model/importProgress';
import { ImportFailure } from './components/ImportFailure/ImportFailure';
import { ImportProgress } from './components/ImportProgress/ImportProgress';
import { ImportReport } from './components/ImportReport/ImportReport';

/** /productos/importar/:importId: the progress of a confirmed import and its final report. */
export function ImportStatusPage() {
  const { importId = '' } = useParams();
  const found = useImport(importId);
  return (
    <Page title="Importar productos" description={found.data?.file_name}>
      <QueryState queries={[found]} errorTitle="No pudimos cargar la importación">
        {() => found.data && <ImportStage found={found.data} />}
      </QueryState>
    </Page>
  );
}

function ImportStage({ found }: { found: Import }) {
  switch (importStage(found.status)) {
    case 'running':
      return <ImportProgress found={found} />;
    case 'completed':
      return <ImportReport counts={found.counts} />;
    case 'failed':
      return <ImportFailure importId={found.id} />;
    default:
      return <ImportFailure />;
  }
}
