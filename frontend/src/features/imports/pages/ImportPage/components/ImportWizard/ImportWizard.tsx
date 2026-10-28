import { useState } from 'react';
import type { SelectOption } from '@/shared/ui';
import type { ImportPreview } from '../../../../api/types';
import { PreviewStep } from '../PreviewStep/PreviewStep';
import { UploadForm } from '../UploadForm/UploadForm';

/** The upload, then its preview: confirming leaves for the status screen, and uploading again starts over. */
export function ImportWizard({ locations }: { locations: SelectOption[] }) {
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  if (!preview) return <UploadForm locations={locations} onUploaded={setPreview} />;
  return <PreviewStep preview={preview} onRestart={() => setPreview(null)} />;
}
