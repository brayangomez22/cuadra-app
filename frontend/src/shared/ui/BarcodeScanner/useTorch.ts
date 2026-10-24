import { useState } from 'react';

// The flashlight is not in the TypeScript types of the media constraints yet.
type TorchCapabilities = MediaTrackCapabilities & { torch?: boolean };

/**
 * The flashlight of the camera, for counters with poor light. Only some
 * phones offer it (Chrome on Android); elsewhere it is not available.
 */
export function useTorch(track: MediaStreamTrack | undefined) {
  const [on, setOn] = useState(false);
  const capabilities = track && 'getCapabilities' in track ? (track.getCapabilities() as TorchCapabilities) : {};
  const available = Boolean(track && capabilities.torch);

  function toggle() {
    if (!track) return;
    const next = !on;
    setOn(next);
    track.applyConstraints({ advanced: [{ torch: next } as MediaTrackConstraintSet] }).catch(() => setOn(!next));
  }

  return { available, on, toggle };
}
