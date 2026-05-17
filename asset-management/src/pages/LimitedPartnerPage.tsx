import { useState, useEffect } from 'react';
import '../styles/LimitedPartnerPage.css';

interface LPCall {
  id: string;
  fund: string;
  target: number;
  received: number;
  lpCompletion: string;
  deadlineDate: string;
  status: string;
  commitmentUSD: number;
  drawAmountUSD: number;
  lpStatus: string;
}

interface LimitedPartnerPageProps {
  lpId: string;
  onBack: () => void;
}

// ── Helpers ──────────────────────────────────────────────────────────────────

const fmt = (n: number) =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(n);

const fmtDeadline = (iso: string) => {
  if (!iso) return '—';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: 'numeric' });
};

const daysRemaining = (iso: string): string => {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  const diff = Math.ceil((d.getTime() - Date.now()) / (1000 * 3600 * 24));
  return diff > 0 ? `${diff}d remaining` : 'Past due';
};

const lpStatusBadgeClass = (s: string) => {
  switch (s) {
    case 'committed': return 'lp-badge lp-badge-committed';
    case 'defaulted':  return 'lp-badge lp-badge-defaulted';
    case 'deferred':   return 'lp-badge lp-badge-deferred';
    default:           return 'lp-badge lp-badge-pending';
  }
};

const callStatusBadgeClass = (s: string) => {
  switch (s) {
    case 'settled':  return 'lp-badge lp-badge-committed';   // green
    case 'issued':   return 'lp-badge lp-badge-pending';     // amber
    default:         return 'lp-badge lp-badge-deferred';
  }
};

const avatarInitials = (id: string) =>
  id.replace('lp-', 'LP').toUpperCase().slice(0, 3);

// ── Component ────────────────────────────────────────────────────────────────

export default function LimitedPartnerPage({ lpId, onBack }: LimitedPartnerPageProps) {
  const [calls, setCalls] = useState<LPCall[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        const res = await fetch(`/api/lps/${lpId}/capital-calls`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data: LPCall[] = await res.json();
        if (!cancelled) {
          setCalls(Array.isArray(data) ? data : []);
          setLoading(false);
        }
      } catch (e: any) {
        if (!cancelled) {
          setError(e.message ?? 'Failed to load data');
          setLoading(false);
        }
      }
    };

    load();
    const interval = setInterval(load, 10_000);
    return () => { cancelled = true; clearInterval(interval); };
  }, [lpId]);

  // ── Derived stats ────────────────────────────────────────────────────────
  const totalCommitment = calls.reduce((s, c) => s + c.commitmentUSD, 0);
  const totalDraw = calls.reduce((s, c) => s + c.drawAmountUSD, 0);
  const activeCalls = calls.filter(c => c.status === 'issued').length;
  const completedCalls = calls.filter(c => c.status === 'settled').length;

  return (
    <div className="lp-page">
      {/* ── Navbar ── */}
      <nav className="lp-navbar">
        <div className="lp-nav-brand">Intelligent Capital Call &amp; Liquidity Orchestration</div>
        <button className="lp-nav-back-btn" onClick={onBack}>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24"
            fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="m15 18-6-6 6-6" />
          </svg>
          Back to Login
        </button>
      </nav>

      <main className="lp-main-content">
        {/* ── Header ── */}
        <header className="lp-header">
          <div className="lp-header-left">
            <div className="lp-avatar">{avatarInitials(lpId)}</div>
            <div>
              <h1 className="lp-page-title">{lpId.toUpperCase()}</h1>
              <p className="lp-page-subtitle">LP Investor Console — Capital Call Portfolio</p>
            </div>
          </div>
        </header>

        {/* ── Stats grid ── */}
        <div className="lp-stats-grid">
          <div className="lp-stat-card">
            <div className="lp-stat-label">Total Commitment</div>
            <div className="lp-stat-value" style={{ fontSize: totalCommitment > 1e8 ? '1.35rem' : undefined }}>
              {loading ? '—' : fmt(totalCommitment)}
            </div>
          </div>
          <div className="lp-stat-card">
            <div className="lp-stat-label">Total Draw Called</div>
            <div className="lp-stat-value" style={{ color: '#5b53f9', fontSize: totalDraw > 1e8 ? '1.35rem' : undefined }}>
              {loading ? '—' : fmt(totalDraw)}
            </div>
          </div>
          <div className="lp-stat-card">
            <div className="lp-stat-label">Active Calls</div>
            <div className="lp-stat-value" style={{ color: '#d97706' }}>
              {loading ? '—' : String(activeCalls).padStart(2, '0')}
            </div>
          </div>
          <div className="lp-stat-card">
            <div className="lp-stat-label">Completed Calls</div>
            <div className="lp-stat-value" style={{ color: '#16a34a' }}>
              {loading ? '—' : String(completedCalls).padStart(2, '0')}
            </div>
          </div>
        </div>

        {/* ── Table ── */}
        <div className="lp-table-container">
          <div className="lp-table-header">
            <h2 className="lp-table-title">Capital Call Participations</h2>
            {!loading && (
              <span className="lp-table-count">{calls.length} call{calls.length !== 1 ? 's' : ''}</span>
            )}
          </div>

          {loading ? (
            <div className="lp-loading">Loading your capital calls…</div>
          ) : error ? (
            <div className="lp-empty-state">
              <p style={{ color: '#ef4444' }}>Error: {error}</p>
            </div>
          ) : calls.length === 0 ? (
            <div className="lp-empty-state">
              <div className="lp-empty-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24"
                  fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                </svg>
              </div>
              <h3>No Capital Calls Found</h3>
              <p>{lpId.toUpperCase()} has not participated in any capital calls yet.</p>
            </div>
          ) : (
            <div className="lp-table-wrapper">
              <table className="lp-table">
                <thead>
                  <tr>
                    <th>Call ID</th>
                    <th>Fund</th>
                    <th>Target Amount</th>
                    <th>My Commitment</th>
                    <th>Draw Called</th>
                    <th>Deadline</th>
                    <th>Call Status</th>
                    <th>My Status</th>
                  </tr>
                </thead>
                <tbody>
                  {calls.map(call => (
                    <LPCallRow key={call.id} call={call} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}

// ── Row sub-component ─────────────────────────────────────────────────────────

function LPCallRow({ call }: { call: LPCall }) {
  const deadlineStr = fmtDeadline(call.deadlineDate);
  const remaining  = daysRemaining(call.deadlineDate);
  const isPastDue  = remaining === 'Past due';

  return (
    <tr>
      <td>
        <div style={{ fontWeight: 600, color: '#111827', fontSize: '0.9rem' }}>{call.id}</div>
      </td>
      <td style={{ color: '#6b7280', fontSize: '0.875rem' }}>{call.fund}</td>
      <td style={{ fontWeight: 500, color: '#374151' }}>{fmt(call.target)}</td>
      <td style={{ fontWeight: 600, color: '#5b53f9' }}>{fmt(call.commitmentUSD)}</td>
      <td style={{ fontWeight: 500, color: '#374151' }}>
        {call.drawAmountUSD > 0 ? fmt(call.drawAmountUSD) : <span style={{ color: '#9ca3af' }}>—</span>}
      </td>
      <td>
        <div style={{ fontWeight: 500, color: '#111827', fontSize: '0.875rem' }}>{deadlineStr}</div>
        {remaining && (
          <div style={{ fontSize: '0.75rem', color: isPastDue ? '#dc2626' : '#6b7280', marginTop: '2px' }}>
            {remaining}
          </div>
        )}
      </td>
      <td>
        <span className={callStatusBadgeClass(call.status)}>
          {call.status === 'issued' ? 'ACTIVE' : call.status.toUpperCase()}
        </span>
      </td>
      <td>
        <span className={lpStatusBadgeClass(call.lpStatus)}>
          {call.lpStatus.toUpperCase()}
        </span>
      </td>
    </tr>
  );
}
