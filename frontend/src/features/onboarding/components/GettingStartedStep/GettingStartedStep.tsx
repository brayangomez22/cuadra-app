import { Badge, ButtonLink, IconAlert, IconCheck } from '@/shared/ui';
import type { ChecklistStep } from '../../api/types';
import { stepContent } from '../../model/stepContent';
import styles from './GettingStartedStep.module.scss';

type Props = { step: ChecklistStep };

/** A step of the checklist: what to do and the way there, or that it is done. */
export function GettingStartedStep({ step }: Props) {
  const content = stepContent[step.id];
  return (
    <li className={styles.root} data-done={step.done}>
      <span className={styles.mark} aria-hidden="true">
        {step.done && <IconCheck />}
      </span>
      <div className={styles.body}>
        <p className={styles.title}>{content.title}</p>
        {!step.done && <p className={styles.description}>{content.description}</p>}
      </div>
      <div className={styles.status}>
        {step.done && (
          <Badge tone="success" icon={<IconCheck />}>
            Listo
          </Badge>
        )}
        {!step.done && step.critical && (
          <Badge tone="warning" icon={<IconAlert />}>
            Importante
          </Badge>
        )}
        {!step.done && content.action && (
          <ButtonLink to={content.action.to} variant="secondary">
            {content.action.label}
          </ButtonLink>
        )}
      </div>
    </li>
  );
}
