import clsx from 'clsx';
import { useRef, useState } from 'react';
import { Alert } from '../Alert';
import { Button } from '../Button';
import { IconFlashlight } from '../icons';
import { CameraVideo } from './CameraVideo';
import { ManualCode } from './ManualCode';
import styles from './BarcodeScanner.module.scss';
import { scanFeedback } from './scanFeedback';
import { scannerProblem, scannerProblems } from './scannerProblems';
import { useBarcodeDetection } from './useBarcodeDetection';
import { useCamera } from './useCamera';
import { useTorch } from './useTorch';

type Props = {
  /** A code read by the camera (once while it stays in view) or typed. */
  onScan: (code: string) => void;
  className?: string;
};

/**
 * Reads barcodes (EAN-13, EAN-8, Code 128, UPC) and QR codes with the camera
 * of a phone, tablet or PC while it is mounted: mount it in a dialog and the
 * camera turns off when it closes. Each code beeps, vibrates and is announced.
 * When the camera cannot be used (no permission, no camera, no HTTPS) it says
 * why; the code can always be typed. Needs HTTPS, as every camera on the web.
 */
export function BarcodeScanner({ onScan, className }: Props) {
  const video = useRef<HTMLVideoElement>(null);
  const camera = useCamera();
  const torch = useTorch(camera.track);
  const [lastCode, setLastCode] = useState<string>();
  const read = (code: string) => {
    setLastCode(code);
    onScan(code);
  };
  const detection = useBarcodeDetection({
    video,
    active: camera.status === 'ready',
    onScan: (code) => {
      scanFeedback();
      read(code);
    },
  });
  const problem = scannerProblem(camera.status, detection.failed);

  return (
    <div className={clsx(styles.root, className)}>
      {problem ? (
        <Alert tone="danger" title={scannerProblems[problem].title}>
          {scannerProblems[problem].description}
        </Alert>
      ) : (
        <>
          <div className={styles.viewport}>
            <CameraVideo ref={video} stream={camera.stream} className={styles.video} />
            <div className={styles.frame} aria-hidden="true" />
            {torch.available && (
              <Button
                variant="secondary"
                icon={<IconFlashlight />}
                aria-pressed={torch.on}
                onClick={torch.toggle}
                className={styles.torch}
              >
                Linterna
              </Button>
            )}
          </div>
          <p role="status" className={styles.status}>
            {statusText(camera.status, lastCode)}
          </p>
        </>
      )}
      <ManualCode onCode={read} />
    </div>
  );
}

function statusText(status: string, lastCode: string | undefined) {
  if (lastCode) return `Código leído: ${lastCode}`;
  return status === 'ready' ? 'Apunte la cámara al código' : 'Abriendo la cámara…';
}
