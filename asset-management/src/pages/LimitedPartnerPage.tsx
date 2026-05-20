import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import '../styles/LimitedPartnerPage.css';
import LPContributionModal from '../components/LPContributionModal';

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

// lpStatusBadgeClass maps the three canonical DB statuses to badge styles.
// Canonical statuses: "pending" | "committed" | "defaulted"
// "under_review" is a transient UI-only state for risky LPs awaiting GP review;
// it is never persisted as a canonical DB status.
const lpStatusBadgeClass = (s: string) => {
  switch (s) {
    case 'committed':    return 'lp-badge lp-badge-committed';
    case 'defaulted':   return 'lp-badge lp-badge-defaulted';
    case 'under_review': return 'lp-badge lp-badge-pending'; // transient UI-only state
    default:            return 'lp-badge lp-badge-pending';  // covers "pending"
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

export default function LimitedPartnerPage() {
  const { lpId } = useParams() as { lpId: string };
  const navigate = useNavigate();
  const [calls, setCalls] = useState<LPCall[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedCall, setSelectedCall] = useState<LPCall | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const [activeTab, setActiveTab] = useState('All Calls');
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 15;

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        const res = await fetch(`/api/capital-calls?lpId=${lpId}`);
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
  }, [lpId, refreshTrigger]);

  // ── Derived stats ────────────────────────────────────────────────────────
  const totalCommitment = calls.reduce((s, c) => s + c.commitmentUSD, 0);
  const totalDraw = calls.reduce((s, c) => s + c.drawAmountUSD, 0);
  const activeCalls = calls.filter(c => c.status === 'issued').length;
  const completedCalls = calls.filter(c => c.status === 'settled').length;

  // 1. Filter
  const filteredCalls = calls.filter(call => {
    if (activeTab === 'Active') return call.status === 'issued';
    if (activeTab === 'Settled') return call.status === 'settled';
    return true;
  });

  // 2. Search
  const searchedCalls = filteredCalls.filter(call => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return call.id.toLowerCase().includes(q) || 
           call.fund.toLowerCase().includes(q) || 
           call.status.toLowerCase().includes(q) ||
           call.lpStatus.toLowerCase().includes(q);
  });

  // 3. Paginate
  const totalPages = Math.ceil(searchedCalls.length / itemsPerPage) || 1;
  const validPage = Math.min(currentPage, totalPages);
  
  const startIndex = (validPage - 1) * itemsPerPage;
  const paginatedCalls = searchedCalls.slice(startIndex, startIndex + itemsPerPage);

  return (
    <div className="lp-page">
      {/* ── Navbar ── */}
      <nav className="lp-navbar">
        <div className="lp-nav-brand">Intelligent Capital Call &amp; Liquidity Orchestration</div>
        
        <div className="lp-navbar-actions">
          <div className="cc-search-container lp-search-container">
            <svg style={{ position: 'absolute', left: '1rem', top: '50%', transform: 'translateY(-50%)', color: '#9ca3af' }} xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
            <input 
              type="text" 
              placeholder="Search calls..." 
              style={{ width: '100%', padding: '0.5rem 1rem 0.5rem 2.25rem', border: '1px solid #e5e7eb', borderRadius: '6px', fontSize: '0.875rem' }}
              value={searchQuery}
              onChange={e => {
                setSearchQuery(e.target.value);
                setCurrentPage(1);
              }}
            />
          </div>
          <button className="lp-nav-back-btn" onClick={() => navigate('/')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24"
              fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="m15 18-6-6 6-6" />
            </svg>
            Log Out
          </button>
        </div>
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
          <div className="lp-table-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
              <h2 className="lp-table-title">Capital Call Participations</h2>
              {!loading && (
                <span className="lp-table-count">{calls.length} call{calls.length !== 1 ? 's' : ''}</span>
              )}
            </div>
            
            <div className="cc-table-tabs" style={{ padding: 0, borderBottom: 'none', display: 'flex', gap: '1rem' }}>
              {['All Calls', 'Active', 'Settled'].map(tab => (
                <button 
                  key={tab} 
                  className={`cc-tab ${activeTab === tab ? 'active' : ''}`}
                  onClick={() => {
                    setActiveTab(tab);
                    setCurrentPage(1);
                  }}
                  style={{ padding: '0.5rem', marginBottom: 0 }}
                >
                  {tab}
                </button>
              ))}
            </div>
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
          ) : searchedCalls.length === 0 ? (
            <div className="lp-empty-state">
              <p>No capital calls match your filter criteria.</p>
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
                    <th style={{ textAlign: 'right' }}>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {paginatedCalls.map(call => (
                    <LPCallRow key={call.id} call={call} onRespond={() => setSelectedCall(call)} />
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {searchedCalls.length > 0 && !loading && (
            <div className="cc-pagination" style={{ borderTop: '1px solid #e5e7eb', padding: '1rem' }}>
              <div className="cc-pagination-info">
                Showing {startIndex + 1}-{Math.min(startIndex + itemsPerPage, searchedCalls.length)} of {searchedCalls.length} Capital Calls
              </div>
              <div className="cc-pagination-controls">
                <button 
                  className="cc-page-btn cc-page-arrow" 
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={validPage === 1}
                  style={{ opacity: validPage === 1 ? 0.5 : 1, cursor: validPage === 1 ? 'not-allowed' : 'pointer' }}
                >
                  &lt;
                </button>
                
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                  <button 
                    key={page} 
                    className={`cc-page-btn ${validPage === page ? 'active' : ''}`}
                    onClick={() => setCurrentPage(page)}
                  >
                    {page}
                  </button>
                ))}

                <button 
                  className="cc-page-btn cc-page-arrow"
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={validPage === totalPages}
                  style={{ opacity: validPage === totalPages ? 0.5 : 1, cursor: validPage === totalPages ? 'not-allowed' : 'pointer' }}
                >
                  &gt;
                </button>
              </div>
            </div>
          )}
        </div>
      </main>

      {selectedCall && (
        <LPContributionModal
          isOpen={true}
          onClose={() => setSelectedCall(null)}
          onSuccess={() => {
            setSelectedCall(null);
            setRefreshTrigger(prev => prev + 1);
          }}
          callId={selectedCall.id}
          lpId={lpId}
          fundName={selectedCall.fund}
          targetAmount={selectedCall.target}
          drawAmountUSD={selectedCall.drawAmountUSD}
          deadlineDate={selectedCall.deadlineDate}
        />
      )}
    </div>
  );
}

// ── Row sub-component ─────────────────────────────────────────────────────────

function LPCallRow({ call, onRespond }: { call: LPCall, onRespond: () => void }) {
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
      <td style={{ textAlign: 'right' }}>
        {call.lpStatus === 'pending' && call.status === 'issued' ? (
          <button 
            onClick={onRespond}
            style={{
              background: '#5b53f9',
              color: 'white',
              border: 'none',
              padding: '6px 12px',
              borderRadius: '6px',
              cursor: 'pointer',
              fontSize: '0.875rem',
              fontWeight: 500
            }}
          >
            Respond
          </button>
        ) : (
          <span style={{ color: '#9ca3af', fontSize: '0.875rem' }}>—</span>
        )}
      </td>
    </tr>
  );
}
