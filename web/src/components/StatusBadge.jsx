import React from 'react';
import { useTranslation } from 'react-i18next';

const TONES = {
  draft: 'neutral', registration_open: 'info', registration_closed: 'warning', running: 'accent', finished: 'success',
  cancelled: 'danger', archived: 'neutral',
  scheduled: 'neutral', queued: 'info', failed: 'danger',
  created: 'neutral', reserved: 'info', committed: 'success', timed_out: 'danger', aborted: 'neutral',
  enrolled: 'success', withdrawn: 'neutral', disqualified: 'danger',
  validating: 'info', ready: 'success', rejected: 'danger', disabled: 'neutral',
  active: 'success', suspended: 'warning',
  healthy: 'success', degraded: 'warning', down: 'danger', unknown: 'neutral',
  win: 'success', loss: 'neutral', draw: 'info',
  published: 'success', staging: 'info', publish_failed: 'danger',
};

/** Badge for any canonical state. The key forces a re-mount (and the CSS
 * state-change animation) only when the state actually changes. */
export function StatusBadge({ state }) {
  const { t } = useTranslation('common');
  const tone = TONES[state] || 'neutral';
  return (
    <span key={state} className={`badge badge-${tone} badge-animated`} data-state={state}>
      {t(`states.${state}`, { defaultValue: state })}
    </span>
  );
}
