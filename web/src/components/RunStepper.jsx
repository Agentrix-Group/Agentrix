import React from 'react';
import { useTranslation } from 'react-i18next';

const STEPS = ['scheduled', 'queued', 'running', 'finished'];

/** Visual progress of a match: scheduled → queued → running → finished/failed.
 * It only reflects states received from the backend. */
export function RunStepper({ state }) {
  const { t } = useTranslation('common');
  const terminal = state === 'failed' || state === 'cancelled' ? state : null;
  const current = terminal ? STEPS.indexOf('running') : STEPS.indexOf(state);
  return (
    <ol className="stepper" aria-label={t('labels.progress')}>
      {STEPS.map((step, index) => {
        let status = 'todo';
        if (index < current || state === 'finished') status = 'done';
        else if (index === current) status = terminal ? 'error' : 'current';
        const label = index === STEPS.length - 1 && terminal ? terminal : step;
        return (
          <li key={step} className={`step step-${status}`} aria-current={status === 'current' ? 'step' : undefined}>
            <span className="step-dot" aria-hidden="true" />
            <span className="step-label">{t(`states.${label}`)}</span>
          </li>
        );
      })}
    </ol>
  );
}
