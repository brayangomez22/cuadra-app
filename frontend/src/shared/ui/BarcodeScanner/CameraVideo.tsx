import { useEffect, useRef, type Ref } from 'react';

type Props = { stream: MediaStream | undefined; ref: Ref<HTMLVideoElement>; className?: string };

/**
 * The camera's picture. Muted and inline so phones play it without a tap
 * and without going full screen. Decorative: the status says what it sees.
 */
export function CameraVideo({ stream, ref, className }: Props) {
  const own = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const video = own.current;
    if (!video || !stream) return;
    video.srcObject = stream;
    // Fails only if the scanner closes first: nothing left to show.
    video.play().catch(() => undefined);
  }, [stream]);

  return (
    <video
      ref={(element) => {
        own.current = element;
        if (typeof ref === 'function') ref(element);
        else if (ref) ref.current = element;
      }}
      className={className}
      muted
      playsInline
      aria-hidden="true"
    />
  );
}
