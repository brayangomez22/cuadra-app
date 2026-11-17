import { useState } from 'react';
import { useFlag } from '@/features/auth';
import { Button, IconMessage } from '@/shared/ui';
import { FeedbackDialog } from '../FeedbackDialog/FeedbackDialog';

/**
 * "Danos tu opinión": a comment to the Cuadra team, sent with the screen the
 * user is on and the app's version. Shown while the "feedback" flag is on.
 */
export function FeedbackButton({ className }: { className?: string }) {
  const enabled = useFlag('feedback');
  const [open, setOpen] = useState(false);
  if (!enabled) return null;
  return (
    <FeedbackDialog
      open={open}
      onOpenChange={setOpen}
      trigger={
        <Button variant="ghost" icon={<IconMessage />} className={className}>
          Danos tu opinión
        </Button>
      }
    />
  );
}
