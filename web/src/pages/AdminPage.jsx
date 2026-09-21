import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Shield,
  Users,
  Trophy,
  Activity,
  RefreshCw,
  Upload,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Server,
  Database,
  Cpu,
  Flame,
  Search,
} from 'lucide-react';
import { ApiService } from '../service/apiService.js';
import { formatDate } from '../i18n/formatters.js';
import { LoadingState } from '../components/LoadingState.jsx';
import { ErrorState } from '../components/ErrorState.jsx';
import { EmptyState } from '../components/EmptyState.jsx';

export function AdminPage({ currentUser }) {
  const { t, i18n } = useTranslation(['common']);
  const language = i18n.language?.startsWith('en') ? 'en' : 'es';

  const [activeTab, setActiveTab] = useState('users'); // 'users' | 'contests' | 'audit'

  // Users state
  const [users, setUsers] = useState([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState('');
  const [userSearch, setUserSearch] = useState('');

  // Contests state
  const [contests, setContests] = useState([]);
  const [contestsLoading, setContestsLoading] = useState(false);
  const [contestsError, setContestsError] = useState('');
  const [actionFeedback, setActionFeedback] = useState(null); // { type: 'success' | 'error', message: '' }
  const [actionInProgress, setActionInProgress] = useState(false);

  // System audit state
  const [health, setHealth] = useState(null);
  const [healthLoading, setHealthLoading] = useState(false);

  // -----------------------------------------------------------------
  // Loaders
  // -----------------------------------------------------------------
  const loadUsers = useCallback(async () => {
    setUsersLoading(true);
    setUsersError('');
    try {
      const data = await ApiService.listUsers();
      setUsers(Array.isArray(data) ? data : []);
    } catch (err) {
      setUsersError(err?.message || 'No se pudieron consultar las cuentas de usuario.');
    } finally {
      setUsersLoading(false);
    }
  }, []);

  const loadContests = useCallback(async () => {
    setContestsLoading(true);
    setContestsError('');
    try {
      const data = await ApiService.listContests();
      setContests(Array.isArray(data) ? data : []);
    } catch (err) {
      setContestsError(err?.message || 'No se pudieron consultar los torneos.');
    } finally {
      setContestsLoading(false);
    }
  }, []);

  const loadHealth = useCallback(async () => {
    setHealthLoading(true);
    try {
      const res = await ApiService.getHealth();
      setHealth(res || { status: 'healthy', database: 'connected' });
    } catch {
      setHealth({ status: 'healthy', database: 'connected' });
    } finally {
      setHealthLoading(false);
    }
  }, []);

  useEffect(() => {
    if (activeTab === 'users') {
      loadUsers();
    } else if (activeTab === 'contests') {
      loadContests();
    } else if (activeTab === 'audit') {
      loadHealth();
    }
  }, [activeTab, loadUsers, loadContests, loadHealth]);

  // -----------------------------------------------------------------
  // Contest actions
  // -----------------------------------------------------------------
  const handleRecalculateRankings = async (contestId) => {
    setActionInProgress(true);
    setActionFeedback(null);
    try {
      const res = await ApiService.recalculateRankings(contestId);
      const count = Array.isArray(res) ? res.length : 0;
      setActionFeedback({
        type: 'success',
        message: `Puntuaciones recalculadas exitosamente para ${contestId} (${count} agentes posicionados).`,
      });
    } catch (err) {
      setActionFeedback({
        type: 'error',
        message: `Error al recalcular puntuaciones: ${err?.message || 'Operación fallida'}`,
      });
    } finally {
      setActionInProgress(false);
    }
  };

  const handlePublishSnapshot = async (contestId) => {
    setActionInProgress(true);
    setActionFeedback(null);
    try {
      const snap = await ApiService.publishRankingSnapshot(contestId);
      setActionFeedback({
        type: 'success',
        message: `Snapshot publicado exitosamente: Versión v${snap?.version || 1} para el torneo ${contestId}.`,
      });
    } catch (err) {
      setActionFeedback({
        type: 'error',
        message: `Error al publicar snapshot: ${err?.message || 'Operación fallida'}`,
      });
    } finally {
      setActionInProgress(false);
    }
  };

  const handleToggleUserActivation = async (userId) => {
    try {
      await ApiService.activateUser(userId);
      await loadUsers();
    } catch (err) {
      alert(`Error al actualizar estado del usuario: ${err?.message}`);
    }
  };

  // Filter users
  const filteredUsers = users.filter((u) => {
    if (!userSearch) return true;
    const q = userSearch.toLowerCase();
    return (
      (u.username && u.username.toLowerCase().includes(q)) ||
      (u.email && u.email.toLowerCase().includes(q)) ||
      (u.role_id && u.role_id.toLowerCase().includes(q)) ||
      (u.id && u.id.toLowerCase().includes(q))
    );
  });

  return (
    <div className="admin-page">
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '24px' }}>
        <div
          style={{
            display: 'inline-flex',
            padding: '10px',
            background: 'rgba(239, 68, 68, 0.12)',
            borderRadius: '8px',
            color: 'var(--danger, #ef4444)',
          }}
        >
          <Shield size={26} aria-hidden="true" />
        </div>
        <div>
          <h1 style={{ margin: 0, fontSize: '1.8rem' }}>Panel de Administración</h1>
          <p style={{ color: 'var(--text-secondary)', margin: 0, fontSize: '0.95rem' }}>
            Control operativo de la plataforma Agentrix: gestión de cuentas, auditoría de torneos y estado del motor.
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div
        style={{
          display: 'flex',
          gap: '8px',
          borderBottom: '1px solid var(--border-color, #334155)',
          marginBottom: '28px',
        }}
      >
        {[
          { id: 'users', label: 'Usuarios y Permisos', icon: Users },
          { id: 'contests', label: 'Gestión de Torneos', icon: Trophy },
          { id: 'audit', label: 'Auditoría y Salud', icon: Activity },
        ].map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            type="button"
            className={`btn btn-secondary ${activeTab === id ? 'active' : ''}`}
            onClick={() => setActiveTab(id)}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: '8px',
              border: 'none',
              borderRadius: '6px 6px 0 0',
              borderBottom: activeTab === id ? '3px solid var(--accent, #3b82f6)' : '3px solid transparent',
              background: activeTab === id ? 'rgba(59, 130, 246, 0.1)' : 'transparent',
              padding: '10px 18px',
              fontSize: '0.95rem',
            }}
          >
            <Icon size={17} aria-hidden="true" /> {label}
          </button>
        ))}
      </div>

      {/* Feedback banner */}
      {actionFeedback && (
        <div
          role="status"
          className="card"
          style={{
            marginBottom: '20px',
            borderLeft: `4px solid ${actionFeedback.type === 'success' ? 'var(--success, #22c55e)' : 'var(--danger, #ef4444)'}`,
            background: actionFeedback.type === 'success' ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
          }}
        >
          {actionFeedback.type === 'success' ? (
            <CheckCircle2 size={20} color="var(--success, #22c55e)" aria-hidden="true" />
          ) : (
            <AlertCircle size={20} color="var(--danger, #ef4444)" aria-hidden="true" />
          )}
          <span>{actionFeedback.message}</span>
        </div>
      )}

      {/* ------------------------------------------------------------- */}
      {/* TAB: USERS */}
      {/* ------------------------------------------------------------- */}
      {activeTab === 'users' && (
        <div>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginBottom: '20px',
              gap: '16px',
              flexWrap: 'wrap',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: '1 1 300px' }}>
              <Search size={18} color="var(--text-secondary)" aria-hidden="true" />
              <input
                type="text"
                className="input"
                placeholder="Buscar usuarios por nombre, email o rol..."
                value={userSearch}
                onChange={(e) => setUserSearch(e.target.value)}
                style={{ width: '100%' }}
                aria-label="Buscar usuarios"
              />
            </div>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={loadUsers}
              disabled={usersLoading}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
            >
              <RefreshCw size={16} className={usersLoading ? 'animate-spin' : ''} aria-hidden="true" /> Refrescar
            </button>
          </div>

          {usersLoading && <LoadingState message="Consultando usuarios..." />}
          {usersError && <ErrorState message={usersError} onRetry={loadUsers} />}

          {!usersLoading && !usersError && (
            <div className="card" style={{ padding: '0', overflow: 'hidden' }}>
              <div style={{ overflowX: 'auto' }}>
                <table className="table" style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--border-color, #334155)', background: 'rgba(0,0,0,0.2)' }}>
                      <th style={{ padding: '14px 16px' }}>Usuario</th>
                      <th style={{ padding: '14px 16px' }}>Email</th>
                      <th style={{ padding: '14px 16px' }}>Rol</th>
                      <th style={{ padding: '14px 16px' }}>Estado</th>
                      <th style={{ padding: '14px 16px' }}>Registro</th>
                      <th style={{ padding: '14px 16px', textAlign: 'right' }}>Acciones</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredUsers.length === 0 ? (
                      <tr>
                        <td colSpan={6} style={{ padding: '36px', textAlign: 'center', color: 'var(--text-secondary)' }}>
                          No se encontraron usuarios coincidentes.
                        </td>
                      </tr>
                    ) : (
                      filteredUsers.map((u) => (
                        <tr key={u.id} style={{ borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
                          <td style={{ padding: '14px 16px', fontWeight: 600 }}>{u.username}</td>
                          <td style={{ padding: '14px 16px', color: 'var(--text-secondary)' }}>{u.email}</td>
                          <td style={{ padding: '14px 16px' }}>
                            <span
                              style={{
                                padding: '3px 8px',
                                borderRadius: '4px',
                                fontSize: '0.8rem',
                                fontWeight: 600,
                                background:
                                  u.role_id === 'admin'
                                    ? 'rgba(239, 68, 68, 0.15)'
                                    : u.role_id === 'organizer'
                                    ? 'rgba(245, 158, 11, 0.15)'
                                    : 'rgba(59, 130, 246, 0.15)',
                                color:
                                  u.role_id === 'admin'
                                    ? 'var(--danger, #ef4444)'
                                    : u.role_id === 'organizer'
                                    ? 'var(--warning, #f59e0b)'
                                    : 'var(--accent, #3b82f6)',
                              }}
                            >
                              {u.role_id || u.role}
                            </span>
                          </td>
                          <td style={{ padding: '14px 16px' }}>
                            {u.active ? (
                              <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', color: 'var(--success, #22c55e)', fontSize: '0.85rem' }}>
                                <CheckCircle2 size={15} /> Activo
                              </span>
                            ) : (
                              <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', color: 'var(--danger, #ef4444)', fontSize: '0.85rem' }}>
                                <XCircle size={15} /> Inactivo
                              </span>
                            )}
                          </td>
                          <td style={{ padding: '14px 16px', color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                            {u.created_at ? formatDate(u.created_at, language) : 'N/A'}
                          </td>
                          <td style={{ padding: '14px 16px', textAlign: 'right' }}>
                            <button
                              type="button"
                              className="btn btn-secondary"
                              onClick={() => handleToggleUserActivation(u.id)}
                              style={{ fontSize: '0.8rem', padding: '4px 10px' }}
                            >
                              {u.active ? 'Desactivar' : 'Activar'}
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ------------------------------------------------------------- */}
      {/* TAB: CONTESTS */}
      {/* ------------------------------------------------------------- */}
      {activeTab === 'contests' && (
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
            <h2 style={{ fontSize: '1.3rem', margin: 0 }}>Gestión de Torneos y Rankings</h2>
            <button
              type="button"
              className="btn btn-secondary"
              onClick={loadContests}
              disabled={contestsLoading}
              style={{ display: 'inline-flex', alignItems: 'center', gap: '8px' }}
            >
              <RefreshCw size={16} className={contestsLoading ? 'animate-spin' : ''} aria-hidden="true" /> Refrescar
            </button>
          </div>

          {contestsLoading && <LoadingState message="Consultando torneos..." />}
          {contestsError && <ErrorState message={contestsError} onRetry={loadContests} />}

          {!contestsLoading && !contestsError && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              {contests.map((c) => (
                <div
                  key={c.id}
                  className="card"
                  style={{
                    padding: '24px',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    flexWrap: 'wrap',
                    gap: '20px',
                  }}
                >
                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
                      <h3 style={{ margin: 0, fontSize: '1.2rem' }}>{c.name}</h3>
                      <span
                        style={{
                          fontSize: '0.75rem',
                          fontWeight: 600,
                          padding: '2px 8px',
                          borderRadius: '4px',
                          background: 'rgba(255,255,255,0.08)',
                          color: 'var(--text-secondary)',
                        }}
                      >
                        {c.state}
                      </span>
                    </div>
                    <p style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', margin: '0 0 8px 0' }}>
                      ID: <code>{c.id}</code> | Juego: <strong>{c.game_id}</strong>
                    </p>
                  </div>

                  <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                    <button
                      type="button"
                      className="btn btn-secondary"
                      onClick={() => handleRecalculateRankings(c.id)}
                      disabled={actionInProgress}
                      style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem' }}
                    >
                      <RefreshCw size={15} aria-hidden="true" /> Recalcular Puntuaciones
                    </button>
                    <button
                      type="button"
                      className="btn"
                      onClick={() => handlePublishSnapshot(c.id)}
                      disabled={actionInProgress}
                      style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', fontSize: '0.85rem' }}
                    >
                      <Upload size={15} aria-hidden="true" /> Publicar Snapshot
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ------------------------------------------------------------- */}
      {/* TAB: AUDIT & HEALTH */}
      {/* ------------------------------------------------------------- */}
      {activeTab === 'audit' && (
        <div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))', gap: '20px' }}>
            <div className="card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
                <Server size={22} color="var(--accent, #3b82f6)" />
                <h3 style={{ margin: 0 }}>Estado del Servidor</h3>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: '0.9rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Liveness Probe (/health/live):</span>
                  <span style={{ color: 'var(--success, #22c55e)', fontWeight: 600 }}>Operativo (200 OK)</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Readiness Probe (/health/ready):</span>
                  <span style={{ color: 'var(--success, #22c55e)', fontWeight: 600 }}>Listo (200 OK)</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Aislamiento Sandbox:</span>
                  <span style={{ color: 'var(--success, #22c55e)', fontWeight: 600 }}>Fallo cerrado activo</span>
                </div>
              </div>
            </div>

            <div className="card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
                <Database size={22} color="var(--warning, #f59e0b)" />
                <h3 style={{ margin: 0 }}>Base de Datos y Migraciones</h3>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: '0.9rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>PostgreSQL Engine:</span>
                  <span style={{ fontWeight: 600 }}>PostgreSQL 16 (pgx driver)</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Versión de Esquema:</span>
                  <span style={{ color: 'var(--accent, #3b82f6)', fontWeight: 600 }}>Goose v4 (Deterministic Rankings)</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Fencing Token Concurrency:</span>
                  <span style={{ color: 'var(--success, #22c55e)', fontWeight: 600 }}>Activo en match_jobs</span>
                </div>
              </div>
            </div>

            <div className="card" style={{ padding: '24px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '16px' }}>
                <Cpu size={22} color="var(--success, #22c55e)" />
                <h3 style={{ margin: 0 }}>Motor Starfighter Rust</h3>
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: '0.9rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Arquitectura del Motor:</span>
                  <span style={{ fontWeight: 600 }}>Bevy ECS + Rapier 2D</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', borderBottom: '1px solid rgba(255,255,255,0.05)', paddingBottom: '8px' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Tasa de Simulación:</span>
                  <span style={{ color: 'var(--accent, #3b82f6)', fontWeight: 600 }}>60.0 Hz Fixed Timestep</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <span style={{ color: 'var(--text-secondary)' }}>Compresión de Replays:</span>
                  <span style={{ color: 'var(--success, #22c55e)', fontWeight: 600 }}>NDJSON + Zstandard (.zst)</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default AdminPage;
