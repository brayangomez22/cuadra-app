import { useEffect, useEffectEvent, useState, type RefObject } from 'react';
import { createDetector } from './detector';
import { createScanFilter } from './scanFilter';

// How often a frame is read: fast enough to feel instant, light on the battery.
const INTERVAL_MS = 150;
// How long a code must be out of view before it counts again.
const QUIET_MS = 1500;

type Options = {
  video: RefObject<HTMLVideoElement | null>;
  /** Reads only while the camera is showing. */
  active: boolean;
  onScan: (code: string) => void;
};

/**
 * Reads the frames of the video while active and calls onScan once per code
 * (createScanFilter). failed says the reader could not load.
 */
export function useBarcodeDetection({ video, active, onScan }: Options) {
  const [failed, setFailed] = useState(false);
  const scanned = useEffectEvent(onScan);

  useEffect(() => {
    if (!active) return;
    let stopped = false;
    // Read through a function: TypeScript does not see the cleanup change it while a frame is read.
    const running = () => !stopped;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const accept = createScanFilter(QUIET_MS);
    const read = async (detector: Awaited<ReturnType<typeof createDetector>>) => {
      if (!running()) return;
      const frame = video.current;
      // A frame that is not ready yet throws: the next one is read.
      const found = frame ? await detector.detect(frame).catch(() => []) : [];
      const code = accept(
        found.map((barcode) => barcode.rawValue),
        performance.now(),
      );
      if (code && running()) scanned(code);
      if (running()) timer = setTimeout(() => void read(detector), INTERVAL_MS);
    };
    createDetector()
      .then(read)
      .catch(() => {
        if (!stopped) setFailed(true);
      });
    return () => {
      stopped = true;
      clearTimeout(timer);
    };
  }, [active, video]);

  return { failed };
}
