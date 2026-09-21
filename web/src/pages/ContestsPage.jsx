import React, { useCallback, useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Trophy,
  Calendar,
  Search,
  Shield,
  Zap,
  Users,
  Swords,
  ChevronRight,
  ArrowLeft,
  CheckCircle2,
  Clock,
  CircleAlert,
  Flame,
} from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatDate } from '../i18n/formatters.js';
import { LoadingState } from '../components/LoadingState.jsx';
import { ErrorState } from '../components/ErrorState.jsx';
import { EmptyState } from '../components/EmptyState.jsx';
import { EnrollAgentModal } from '../components/EnrollAgentModal.jsx';

export function ContestsPage({ contestId: propContestId, onSelectContest, currentUser, onWatchReplay }) {
  const { t, i18n } = useTranslation(['contests', 'common', 'home', 'rankings']);
  const language = i18n.language?.startsWith('en') ? 'en' : 'es';

  const [contests, setContests] = useState([]);
  const [selectedContestId, setSelectedContestId] = useState(propContestId || null);
  const [status, setStatus] = useState('loading'); // 'loading' | 'error' | 'success'
  const [errorMessage, setErrorMessage] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');

  // Detail view state
  const [contestDetail, setContestDetail] = useState(null);
  const [contestRankings, setContestRankings] = useState([]);
  const [contestEntries, setContestEntries] = useState([]);
  const [detailStatus, setDetailStatus] = useState('idle');
  const [activeTab, setActiveTab] = useState('overview'); // 'overview' | 'leaderboard' | 'rules' | 'entries'

  // Enrollment modal
  const [isEnrollModalOpen, setIsEnrollModalOpen] = useState(false);
  const [enrollSuccessMsg, setEnrollSuccessMsg] = useState('');

  const loadContests = useCallback(async () => {
    setStatus('loading');
    setErrorMessage('');
    try {
      const data = await ApiService.listContests();
      setContests(Array.isArray(data) ? data : []);
      setStatus('success');
    } catch (err) {
      setErrorMessage(err?.message || t('common:messages.operationFailed', 'Error al cargar torneos'));
      setStatus('error');
    }
  }, [t]);

  useEffect(() => {
    loadContests();
  }, [loadContests]);

  // Sync prop changes
  useEffect(() => {
    if (propContestId) {
      setSelectedContestId(propContestId);
    }
  }, [propContestId]);

  // Load contest details when a contest is selected
  const loadContestDetail = useCallback(async (id) => {
    if (!id) return;
    setDetailStatus('loading');
    try {
      const [detail, rankings, entries] = await Promise.all([
        ApiService.getContest(id).catch(() => null),
        ApiService.listRankings(id).catch(() => []),
        ApiService.listContestEntries(id).catch(() => []),
      ]);
      setContestDetail(detail);
      setContestRankings(Array.isArray(rankings) ? rankings : []);
      setContestEntries(Array.isArray(entries) ? entries : []);
      setDetailStatus('success');
    } catch {
      setDetailStatus('error');
    }
  }, []);

  useEffect(() => {
    if (selectedContestId) {
      loadContestDetail(selectedContestId);
    } else {
      setContestDetail(null);
      setDetailStatus('idle');
    }
  }, [selectedContestId, loadContestDetail]);

  const filteredContests = useMemo(() => {
    return contests.filter((c) => {
      const matchesSearch =
        !searchQuery ||
        (c.name && c.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (c.description && c.description.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (c.id && c.id.toLowerCase().includes(searchQuery.toLowerCase()));

      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'active' && (c.state === 'in_progress' || c.state === 'live_final')) ||
        (statusFilter === 'scheduled' && (c.state === 'scheduled' || c.state === 'registration_open')) ||
        (statusFilter === 'completed' && (c.state === 'completed' || c.state === 'archived'));

      return matchesSearch && matchesStatus;
    });
  }, [contests, searchQuery, statusFilter]);

  const handleSelect = (id) => {
    setSelectedContestId(id);
    if (onSelectContest) {
      onSelectContest(id);
    }
  };

  const handleBackToList = () => {
    setSelectedContestId(null);
    if (onSelectContest) {
      onSelectContest(null);
    }
  };

  // State badge styling
  const getStateBadge = (state) => {
    const map = {
      scheduled: { label: 'Programado', color: 'var(--text-secondary, #94a3b8)', bg: 'rgba(148, 163, 184, 0.1)' },
      registration_open: { label: 'Inscripciones Abiertas', color: 'var(--success, #22c55e)', bg: 'rgba(34, 197, 94, 0.1)' },
      in_progress: { label: 'En Progreso', color: 'var(--warning, #f59e0b)', bg: 'rgba(245, 158, 11, 0.1)' },
      completed: { label: 'Finalizado', color: 'var(--accent, #3b82f6)', bg: 'rgba(59, 130, 246, 0.1)' },
    };
    const s = map[state] || { label: state || 'Desconocido', color: 'var(--text-secondary)', bg: 'rgba(255,255,255,0.05)' };
    return (
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: '6px',
          padding: '4px 10px',
          borderRadius: '999px',
          fontSize: '0.8rem',
          fontWeight: 600,
          color: s.color,
          backgroundColor: s.bg,
        }}
      >
        <span style={{ width: '6px', height: '6px', borderRadius: '50%', background: s.color }} />
        {s.label}
      </span>
    );
  };

  // -------------------------------------------------------------
  // DETAIL VIEW
  // -------------------------------------------------------------
  if (selectedContestId) {
    const contest = contestDetail || contests.find((c) => c.id === selectedContestId) || { id: selectedContestId, name: selectedContestId };

    return (
      <div className="contest-detail-page">
        <button
          type="button"
          className="btn btn-secondary"
          onClick={handleBackToList}
          style={{ display: 'inline-flex', alignItems: 'center', gap: '8px', marginBottom: '24px' }}
        >
          <ArrowLeft size={16} aria-hidden="true" /> Volver a todos los torneos
        </button>

        {enrollSuccessMsg && (
          <div
            role="status"
            className="card"
            style={{
              marginBottom: '20px',
              borderLeft: '4px solid var(--success, #22c55e)',
              background: 'rgba(34, 197, 94, 0.1)',
            }}
          >
            {enrollSuccessMsg}
          </div>
        )}

        <div className="card" style={{ padding: '32px', marginBottom: '28px', background: 'var(--card-bg, #1e2430)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
                <h1 style={{ margin: 0, fontSize: '1.8rem' }}>{contest.name}</h1>
                {getStateBadge(contest.state)}
              </div>
              <p style={{ color: 'var(--text-secondary)', margin: '0 0 16px 0', fontSize: '1rem', maxWidth: '640px' }}>
                {contest.description || 'Torneo oficial de combate espacial autónomo Starfighter en Agentrix.'}
              </p>
              <div style={{ display: 'flex', gap: '24px', flexWrap: 'wrap', color: 'var(--text-secondary)', fontSize: '0.9rem' }}>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                  <Swords size={16} color="var(--accent)" /> Juego: <strong>{contest.game_id || 'starfighter'}</strong>
                </span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                  <Calendar size={16} /> Inicio: {contest.start_date ? formatDate(contest.start_date, language) : 'Próximamente'}
                </span>
                <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                  <Users size={16} /> Participantes: <strong>{contestEntries.length}</strong>
                </span>
              </div>
            </div>

            <div style={{ display: 'flex', gap: '10px' }}>
              <button
                type="button"
                className="btn"
                onClick={() => setIsEnrollModalOpen(true)}
                style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
              >
                <Zap size={16} aria-hidden="true" /> Inscribir agente
              </button>
            </div>
          </div>

          {/* Sub-navigation tabs */}
          <div
            style={{
              display: 'flex',
              gap: '12px',
              borderBottom: '1px solid var(--border-color, #334155)',
              marginTop: '28px',
              paddingBottom: '2px',
            }}
          >
            {[
              { id: 'overview', label: 'Resumen' },
              { id: 'leaderboard', label: `Clasificación (${contestRankings.length})` },
              { id: 'rules', label: 'Reglas del juego' },
              { id: 'entries', label: `Agentes inscritos (${contestEntries.length})` },
            ].map((t) => (
              <button
                key={t.id}
                type="button"
                className={`btn btn-secondary ${activeTab === t.id ? 'active' : ''}`}
                onClick={() => setActiveTab(t.id)}
                style={{
                  border: 'none',
                  borderRadius: '6px 6px 0 0',
                  borderBottom: activeTab === t.id ? '3px solid var(--accent, #3b82f6)' : '3px solid transparent',
                  background: activeTab === t.id ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
                }}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* Tab content */}
        {activeTab === 'overview' && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '20px' }}>
            <div className="card" style={{ padding: '24px' }}>
              <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '14px' }}>
                <Trophy size={18} color="var(--accent)" /> Formato de Puntuación
              </h3>
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.92rem', lineHeight: 1.6 }}>
                Sistema determinista auditable: 3 puntos por victoria limpia, 1 punto por empate, 0 por derrota.
                Los empates en tabla se resuelven por orden estricto de: diferencia de daño acumulado, enfrentamiento directo (head-to-head) y victorias totales.
              </p>
            </div>

            <div className="card" style={{ padding: '24px' }}>
              <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '14px' }}>
                <Shield size={18} color="var(--success, #22c55e)" /> Requisitos de Admisión
              </h3>
              <ul style={{ color: 'var(--text-secondary)', fontSize: '0.92rem', lineHeight: 1.7, paddingLeft: '20px' }}>
                <li>Bot implementado en Python 3 con interfaz Starfighter.</li>
                <li>Tiempo máximo por tick de decisión: <strong>16.6 ms (60 Hz)</strong>.</li>
                <li>Aislamiento estricto: sin acceso a sockets ni sistema de archivos.</li>
              </ul>
            </div>
          </div>
        )}

        {activeTab === 'leaderboard' && (
          <div className="card" style={{ padding: '24px' }}>
            <h3 style={{ marginBottom: '16px' }}>Clasificación del Torneo</h3>
            {contestRankings.length === 0 ? (
              <EmptyState
                icon={Trophy}
                title="Aún no hay partidas computadas"
                description="Las posiciones se actualizarán automáticamente tan pronto como se ejecuten las partidas programadas."
              />
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table className="table" style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--border-color, #334155)', color: 'var(--text-secondary)' }}>
                      <th style={{ padding: '12px 8px' }}>#</th>
                      <th style={{ padding: '12px 8px' }}>Agente</th>
                      <th style={{ padding: '12px 8px' }}>Piloto</th>
                      <th style={{ padding: '12px 8px' }}>Puntos</th>
                      <th style={{ padding: '12px 8px' }}>Partidas</th>
                      <th style={{ padding: '12px 8px' }}>V / E / D</th>
                    </tr>
                  </thead>
                  <tbody>
                    {contestRankings.map((r, idx) => (
                      <tr key={r.id || idx} style={{ borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                        <td style={{ padding: '12px 8px', fontWeight: 600 }}>{idx + 1}</td>
                        <td style={{ padding: '12px 8px', fontWeight: 500 }}>{r.agent_id}</td>
                        <td style={{ padding: '12px 8px', color: 'var(--text-secondary)' }}>{r.user_id || 'N/A'}</td>
                        <td style={{ padding: '12px 8px', fontWeight: 700, color: 'var(--accent)' }}>{r.score || 0}</td>
                        <td style={{ padding: '12px 8px' }}>{r.matches_played || 0}</td>
                        <td style={{ padding: '12px 8px', color: 'var(--text-secondary)' }}>
                          {r.wins || 0} / {r.draws || 0} / {r.losses || 0}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {activeTab === 'rules' && (
          <div className="card" style={{ padding: '28px' }}>
            <h3 style={{ marginBottom: '16px' }}>Reglas Oficiales de Simulación: Starfighter</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px', color: 'var(--text-secondary)', fontSize: '0.95rem', lineHeight: 1.6 }}>
              <div style={{ background: 'rgba(0,0,0,0.2)', padding: '18px', borderRadius: '8px' }}>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '8px' }}>⚡ Ciclo de Simulación (60 Hz)</h4>
                <p>
                  El motor Bevy/Rapier avanza en pasos de tiempo fijos y deterministas. Cada tick evalúa el estado actual, despacha percepciones privadas y consume la acción del agente.
                </p>
              </div>
              <div style={{ background: 'rgba(0,0,0,0.2)', padding: '18px', borderRadius: '8px' }}>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '8px' }}>🛡️ Energía y Cañón Láser</h4>
                <p>
                  Cada disparo consume energía del reactor. Las naves cuentan con tasa de recarga continua. Disparar sin energía genera un pulso fallido y penalización térmica.
                </p>
              </div>
              <div style={{ background: 'rgba(0,0,0,0.2)', padding: '18px', borderRadius: '8px' }}>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '8px' }}>🌌 Límites de Arena</h4>
                <p>
                  La arena posee barreras de contención elásticas. Chocar a gran velocidad contra los bordes inflige daño estructural al casco de la nave.
                </p>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'entries' && (
          <div className="card" style={{ padding: '24px' }}>
            <h3 style={{ marginBottom: '16px' }}>Agentes Registrados ({contestEntries.length})</h3>
            {contestEntries.length === 0 ? (
              <EmptyState
                icon={Users}
                title="Sin inscripciones registradas"
                description="Sé el primer piloto en inscribir tu nave en esta competencia."
                actionLabel="Inscribir agente"
                onAction={() => setIsEnrollModalOpen(true)}
              />
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: '16px' }}>
                {contestEntries.map((e) => (
                  <div
                    key={e.id}
                    style={{
                      padding: '16px',
                      borderRadius: '8px',
                      background: 'rgba(255,255,255,0.03)',
                      border: '1px solid var(--border-color, #334155)',
                    }}
                  >
                    <div style={{ fontWeight: 600, fontSize: '1.05rem', marginBottom: '4px' }}>{e.agent_name || e.agent_id}</div>
                    <div style={{ color: 'var(--text-secondary)', fontSize: '0.85rem', marginBottom: '8px' }}>
                      Piloto: {e.username || e.user_id}
                    </div>
                    <span
                      style={{
                        fontSize: '0.75rem',
                        fontWeight: 600,
                        padding: '2px 8px',
                        borderRadius: '4px',
                        background: 'rgba(34, 197, 94, 0.1)',
                        color: 'var(--success, #22c55e)',
                      }}
                    >
                      {e.status || 'Activo'}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {isEnrollModalOpen && (
          <EnrollAgentModal
            contest={contest}
            onClose={() => setIsEnrollModalOpen(false)}
            onEnrolled={() => {
              setIsEnrollModalOpen(false);
              setEnrollSuccessMsg('¡Agente inscrito exitosamente en el torneo!');
              loadContestDetail(contest.id);
            }}
          />
        )}
      </div>
    );
  }

  // -------------------------------------------------------------
  // LIST VIEW
  // -------------------------------------------------------------
  return (
    <div className="contests-page">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px', marginBottom: '24px' }}>
        <div>
          <h1 style={{ margin: '0 0 8px 0', fontSize: '1.8rem' }}>Torneos y Competencias</h1>
          <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
            Explora los torneos activos de Starfighter, consulta reglas oficiales, revisa clasificaciones e inscribe tus agentes.
          </p>
        </div>
      </div>

      {/* Filters and search */}
      <div
        className="card"
        style={{
          padding: '16px 20px',
          marginBottom: '24px',
          display: 'flex',
          gap: '16px',
          flexWrap: 'wrap',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flex: '1 1 300px' }}>
          <Search size={18} color="var(--text-secondary)" aria-hidden="true" />
          <input
            type="text"
            className="input"
            placeholder="Buscar torneos por nombre o descripción..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            style={{ width: '100%' }}
            aria-label="Buscar torneos"
          />
        </div>

        <div style={{ display: 'flex', gap: '8px' }}>
          {[
            { id: 'all', label: 'Todos' },
            { id: 'active', label: 'En progreso' },
            { id: 'scheduled', label: 'Programados' },
            { id: 'completed', label: 'Finalizados' },
          ].map((f) => (
            <button
              key={f.id}
              type="button"
              className={`btn btn-secondary ${statusFilter === f.id ? 'active' : ''}`}
              onClick={() => setStatusFilter(f.id)}
              style={{
                fontSize: '0.85rem',
                padding: '6px 12px',
                background: statusFilter === f.id ? 'var(--accent, #3b82f6)' : undefined,
                color: statusFilter === f.id ? '#fff' : undefined,
              }}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {status === 'loading' && <LoadingState message="Cargando torneos..." />}

      {status === 'error' && (
        <ErrorState
          title="Error al cargar torneos"
          message={errorMessage}
          onRetry={loadContests}
        />
      )}

      {status === 'success' && filteredContests.length === 0 && (
        <EmptyState
          icon={Trophy}
          title="No se encontraron torneos"
          description={
            searchQuery
              ? `No hay torneos que coincidan con "${searchQuery}".`
              : 'No hay torneos disponibles en este momento.'
          }
        />
      )}

      {status === 'success' && filteredContests.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '20px' }}>
          {filteredContests.map((c) => (
            <div
              key={c.id}
              className="card"
              style={{
                padding: '24px',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between',
                transition: 'transform 0.15s ease',
              }}
            >
              <div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
                  <span
                    style={{
                      fontSize: '0.8rem',
                      fontWeight: 600,
                      color: 'var(--accent, #3b82f6)',
                      textTransform: 'uppercase',
                      letterSpacing: '0.05em',
                    }}
                  >
                    {c.game_id || 'starfighter'}
                  </span>
                  {getStateBadge(c.state)}
                </div>

                <h3 style={{ margin: '0 0 8px 0', fontSize: '1.25rem' }}>{c.name}</h3>
                <p
                  style={{
                    color: 'var(--text-secondary)',
                    fontSize: '0.92rem',
                    margin: '0 0 20px 0',
                    lineHeight: 1.5,
                    display: '-webkit-box',
                    WebkitLineClamp: 3,
                    WebkitBoxOrient: 'vertical',
                    overflow: 'hidden',
                  }}
                >
                  {c.description || 'Competencia oficial en Starfighter.'}
                </p>
              </div>

              <div style={{ borderTop: '1px solid var(--border-color, #334155)', paddingTop: '16px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', fontSize: '0.85rem', color: 'var(--text-secondary)' }}>
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                    <Calendar size={15} /> {c.start_date ? formatDate(c.start_date, language) : 'Próximamente'}
                  </span>
                </div>

                <div style={{ display: 'flex', gap: '10px' }}>
                  <button
                    type="button"
                    className="btn"
                    onClick={() => handleSelect(c.id)}
                    style={{ flex: 1, display: 'inline-flex', justifyContent: 'center', alignItems: 'center', gap: '6px' }}
                  >
                    Ver detalles <ChevronRight size={16} aria-hidden="true" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ContestsPage;
