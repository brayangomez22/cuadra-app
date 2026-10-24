import { useEffect, useState } from 'react';

export type CameraStatus = 'starting' | 'ready' | 'denied' | 'no_camera' | 'insecure' | 'failed';

// The back camera of a phone or tablet; a PC uses the one it has.
const CONSTRAINTS: MediaStreamConstraints = { video: { facingMode: { ideal: 'environment' } }, audio: false };

// The reasons getUserMedia gives (DOMException names) that the cashier can act on.
const statusByError: Record<string, CameraStatus | undefined> = {
  NotAllowedError: 'denied',
  SecurityError: 'denied',
  NotFoundError: 'no_camera',
  OverconstrainedError: 'no_camera',
};

// Browsers expose mediaDevices only over HTTPS (or localhost).
function initialStatus(): CameraStatus {
  if ('mediaDevices' in navigator) return 'starting';
  return window.isSecureContext ? 'no_camera' : 'insecure';
}

function statusOf(error: unknown): CameraStatus {
  const known = error instanceof DOMException ? statusByError[error.name] : undefined;
  return known ?? 'failed';
}

/**
 * Opens the camera while the component is mounted and turns it off when it
 * unmounts (the camera light goes out). The stream goes to a video element;
 * its track is there for the flashlight.
 */
export function useCamera() {
  const [status, setStatus] = useState(initialStatus);
  const [stream, setStream] = useState<MediaStream>();

  useEffect(() => {
    if (!('mediaDevices' in navigator)) return;
    let opened: MediaStream | undefined;
    let closed = false;
    navigator.mediaDevices
      .getUserMedia(CONSTRAINTS)
      .then((camera) => {
        // Closed while the browser asked for permission: turn it off at once.
        if (closed) {
          stopTracks(camera);
          return;
        }
        opened = camera;
        setStream(camera);
        setStatus('ready');
      })
      .catch((error: unknown) => {
        if (!closed) setStatus(statusOf(error));
      });
    return () => {
      closed = true;
      if (opened) stopTracks(opened);
    };
  }, []);

  return { status, stream, track: stream?.getVideoTracks()[0] };
}

function stopTracks(stream: MediaStream) {
  stream.getTracks().forEach((track) => track.stop());
}
