import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowRight, CalendarDays, CircleAlert, RefreshCw, Sparkles } from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatDate } from '../i18n/formatters.js';

export function HomePage() {
  const { t, i18n } = useTranslation('home');
  const language = i18n.language?.startsWith('en') ? 'en' : 'es';
  const [contests, setContests] = useState([]);
  const [status, setStatus] = useState('loading');

  const loadContests = useCallback(async () => {
    setStatus('loading');
    try {
      const result = await ApiService.listContests();
      if (!Array.isArray(result)) {
        throw new Error('Invalid public contests response');
      }
      setContests(result);
      setStatus('success');
    } catch {
      setStatus('error');
    }
  }, []);

  useEffect(() => {
    loadContests();
  }, [loadContests]);

  const renderDate = (value) => value ? (
    <time dateTime={value}>{formatDate(value, language)}</time>
  ) : t('datePending');

  return (
    <div className="home-page">
      <section className="home-hero" aria-labelledby="home-title">
        <div className="home-hero-content">
          <span className="eyebrow"><Sparkles size={16} aria-hidden="true" /> {t('eyebrow')}</span>
          <h1 id="home-title">{t('title')}</h1>
          <p>{t('subtitle')}</p>
          <div className="home-hero-actions">
            <a className="btn" href="#contests">
              {t('explore')} <ArrowRight size={18} aria-hidden="true" />
            </a>
          </div>
        </div>
        <div className="home-hero-visual" aria-hidden="true">
          <div className="orbit orbit-outer" />
          <div className="orbit orbit-inner" />
          <div className="orbit-core"><span>A</span></div>
          <span className="orbit-dot orbit-dot-one" />
          <span className="orbit-dot orbit-dot-two" />
          <span className="orbit-dot orbit-dot-three" />
        </div>
      </section>

      <section className="home-contests" id="contests" aria-labelledby="contests-title" aria-busy={status === 'loading'}>
        <div className="section-heading">
          <div>
            <span className="section-kicker">{t('sectionKicker')}</span>
            <h2 id="contests-title">{t('contestsTitle')}</h2>
            <p>{t('contestsDescription')}</p>
          </div>
          {status === 'success' && (
            <button className="btn btn-subtle" type="button" onClick={loadContests}>
              <RefreshCw size={17} aria-hidden="true" /> {t('refresh')}
            </button>
          )}
        </div>

        {status === 'loading' && (
          <div className="state-panel" role="status">
            <span className="loading-indicator" aria-hidden="true" />
            <p>{t('loading')}</p>
          </div>
        )}

        {status === 'error' && (
          <div className="state-panel" role="alert">
            <CircleAlert size={28} aria-hidden="true" />
            <h3>{t('errorTitle')}</h3>
            <p>{t('errorDescription')}</p>
            <button className="btn btn-secondary" type="button" onClick={loadContests}>
              <RefreshCw size={17} aria-hidden="true" /> {t('retry')}
            </button>
          </div>
        )}

        {status === 'success' && contests.length === 0 && (
          <div className="state-panel" role="status">
            <CalendarDays size={30} aria-hidden="true" />
            <h3>{t('emptyTitle')}</h3>
            <p>{t('emptyDescription')}</p>
          </div>
        )}

        {status === 'success' && contests.length > 0 && (
          <div className="contest-grid">
            {contests.map((contest) => (
              <article className="contest-card" key={contest.id}>
                <div className="contest-card-top">
                  <span className="contest-symbol" aria-hidden="true"><Sparkles size={20} /></span>
                  <span className={`contest-status contest-status-${contest.state}`}>
                    {t(`status.${contest.state}`, { defaultValue: contest.state })}
                  </span>
                </div>
                <h3>{contest.name}</h3>
                <p className="contest-description">{contest.description || t('noDescription')}</p>
                <div className="contest-dates">
                  <CalendarDays size={18} aria-hidden="true" />
                  <span>{t('startsAt')}: {renderDate(contest.starts_at)}</span>
                  <span>{t('endsAt')}: {renderDate(contest.ends_at)}</span>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
