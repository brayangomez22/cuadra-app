import { useEffect, useRef } from 'react';
import { useBlocker } from 'react-router';
import { Button, Dialog } from '@/shared/ui';

type Props = {
  /** Read when the user leaves: true while there is something unsaved. */
  shouldBlock: () => boolean;
  title: string;
  description: string;
};

/**
 * Asks before leaving a form with unsaved work: a dialog for a link of the
 * app and the browser's own warning for a reload or closing the tab. It
 * renders nothing while nobody leaves.
 */
export function DiscardChangesGuard({ shouldBlock, title, description }: Props) {
  const check = useRef(shouldBlock);
  useEffect(() => {
    check.current = shouldBlock;
  });
  const blocker = useBlocker(() => check.current());

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => {
      if (check.current()) event.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, []);

  const blocked = blocker.state === 'blocked';
  return (
    <Dialog
      open={blocked}
      onOpenChange={(open) => !open && blocked && blocker.reset()}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={() => blocked && blocker.reset()}>
            Seguir editando
          </Button>
          <Button variant="danger" onClick={() => blocked && blocker.proceed()}>
            Descartar
          </Button>
        </>
      }
    />
  );
}
