import { useAuth } from '@/features/auth';
import { Button, Card } from '@/shared/ui';
import { useChecklist, useDismissChecklist } from '../../api/checklist';
import { useStepCompletedEvents } from '../../hooks/useStepCompletedEvents';
import { GettingStartedStep } from '../GettingStartedStep/GettingStartedStep';
import styles from './GettingStarted.module.scss';

/**
 * The getting-started checklist of a new tenant, on the home screen: the
 * steps the user can resolve, done only once their data exists. It shows
 * while the API says so; hidden, it comes back only if a critical step goes
 * missing. While it loads or if it fails it shows nothing: it is a guide,
 * not the screen's content.
 */
export function GettingStarted() {
  const auth = useAuth();
  const checklist = useChecklist();
  const dismiss = useDismissChecklist();
  const tenantId = auth.status === 'authenticated' ? auth.user.tenant_id : '';
  useStepCompletedEvents(tenantId, checklist.data?.steps);

  const data = checklist.data;
  if (!data?.visible) return null;
  return (
    <Card
      title="Primeros pasos"
      className={styles.root}
      action={
        <Button
          variant="ghost"
          aria-label="Ocultar primeros pasos"
          loading={dismiss.isPending}
          onClick={() => dismiss.mutate()}
        >
          Ocultar
        </Button>
      }
    >
      <p className={styles.progress}>{`${String(data.completed)} de ${String(data.total)} listos`}</p>
      <div className={styles.bar} aria-hidden="true">
        {data.steps.map((step) => (
          <span key={step.id} className={styles.segment} data-done={step.done} />
        ))}
      </div>
      <ol className={styles.steps}>
        {data.steps.map((step) => (
          <GettingStartedStep key={step.id} step={step} />
        ))}
      </ol>
    </Card>
  );
}
