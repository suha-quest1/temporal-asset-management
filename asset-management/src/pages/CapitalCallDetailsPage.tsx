import { useState, useEffect } from 'react';
import '../styles/CapitalCallDetailsPage.css';
import '../styles/LPRiskTrackerPage.css';
import { getRiskyLPs, postGPDecision, getCallLPs, getCapitalCalls, postForceSettlement, postCancelCall, getCallTimeline } from '../api/CapitalCall';
import type { CapitalCall } from '../components/CapitalCallRow';

import { useParams, useNavigate } from 'react-router-dom';

interface RiskyLP {
  callId: string;
  lpId: string;
  riskScore: number;
  commitmentUSD: number;
  drawAmountUSD: number;
  lpStatus: string;
  callStatus: string;
  flaggedAt: string;
}

// Raw row returned by /api/capital-calls/:callId/lps
interface CallLP {
  lpId: string;
  commitmentUSD: number;
  drawAmountUSD: number;
  status: string;
  riskScore: number | null;
}

const currencyFmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });

// Timeline event shape returned by GET /api/capital-calls/:callId/timeline
interface TimelineEvent {
  name: string;
  title: string;
  status: string;
  timestamp: string;
  color: string;
}

export default function CapitalCallDetailsPage() {
  const { callId } = useParams();
  const navigate = useNavigate();
  // All LP rows for this call (base ledger)
  const [callLPs, setCallLPs] = useState<CallLP[]>([]);
  // Risky LPs overlay (only high-risk LPs, used to enrich rows)
  const [riskyLPs, setRiskyLPs] = useState<RiskyLP[]>([]);
  // GP decisions made in this session (persist across polling cycles)
  const [reviewedLPs, setReviewedLPs] = useState<Record<string, 'waived' | 'enforced'>>({});
  const [selectedLP, setSelectedLP] = useState<RiskyLP | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [lpLoading, setLpLoading] = useState(true);
  const [portfolioRisk, setPortfolioRisk] = useState<any>(null);

  // Local state for the call
  const [activeCall, setActiveCall] = useState<CapitalCall | null>(null);

  // Live timeline events from Temporal workflow history
  const [timelineEvents, setTimelineEvents] = useState<TimelineEvent[]>([]);

  // Fetch the latest call details dynamically
  useEffect(() => {
    if (!callId) return;
    const fetchCall = () => {
      getCapitalCalls()
        .then(calls => {
          const updated = calls.find((c: CapitalCall) => c.id === callId);
          if (updated) {
            setActiveCall(updated);
          }
        })
        .catch(err => console.error('Failed to sync capital call state', err));
    };
    
    fetchCall();
    const interval = setInterval(fetchCall, 5000);
    return () => clearInterval(interval);
  }, [callId]);

  // Fetch live workflow timeline from Temporal history
  useEffect(() => {
    if (!callId) return;
    const fetchTimeline = () => {
      getCallTimeline(callId)
        .then((data: TimelineEvent[]) => setTimelineEvents(Array.isArray(data) ? data : []))
        .catch(err => console.error('Failed to fetch timeline', err));
    };
    fetchTimeline();
    const interval = setInterval(fetchTimeline, 5000);
    return () => clearInterval(interval);
  }, [callId]);

  // Fetch all LPs for this call (base ledger — always shown)
  useEffect(() => {
    if (!callId) return;
    getCallLPs(callId)
      .then((data: CallLP[]) => {
        setCallLPs(data);
        setLpLoading(false);
      })
      .catch(err => {
        console.error('Failed to fetch call LPs', err);
        setLpLoading(false);
      });
    // Poll every 5s for live status updates
    const interval = setInterval(() => {
      getCallLPs(callId)
        .then((data: CallLP[]) => setCallLPs(data))
        .catch(() => {});
    }, 5000);
    return () => clearInterval(interval);
  }, [callId]);

  // Fetch risky LP overlay (used to add risk badge + review button)
  useEffect(() => {
    if (!callId) return;
    const fetch = () => {
      getRiskyLPs()
        .then((data: RiskyLP[]) => setRiskyLPs(data.filter(lp => lp.callId === callId)))
        .catch(err => console.error('Failed to fetch risky LPs', err));
    };
    fetch();
    const interval = setInterval(fetch, 5000);
    return () => clearInterval(interval);
  }, [callId]);

  // Build an index of risky LPs for O(1) lookup
  const riskyLPMap: Record<string, RiskyLP> = {};
  riskyLPs.forEach(lp => { riskyLPMap[lp.lpId] = lp; });

  const handleAction = async (action: 'waive' | 'enforce') => {
    if (!selectedLP || submitting) return;
    setSubmitting(true);
    try {
      await postGPDecision(selectedLP.callId, {
        lpId: selectedLP.lpId,
        action,
        gpName: 'GP Admin',
      });
      // Record the review outcome — persists for the session, rows stay visible
      setReviewedLPs(prev => ({ ...prev, [selectedLP.lpId]: action === 'waive' ? 'waived' : 'enforced' }));
      setSelectedLP(null);
    } catch (err) {
      console.error(`Failed to ${action} LP`, err);
      alert(`Failed to execute action: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  useEffect(() => {
    if (activeCall?.status === 'settled') {
      fetch(`/reports/${activeCall.id}.json`)
        .then(res => res.json())
        .then(data => {
          if (data.portfolioRisk) {
            setPortfolioRisk(data.portfolioRisk);
          }
        })
        .catch(err => console.log("Report not available yet or fetch failed", err));
    }
  }, [activeCall?.status, activeCall?.id]);

  const handleForceSettlement = async () => {
    if (!activeCall || submitting) return;
    setSubmitting(true);
    try {
      await postForceSettlement(activeCall.id);
      alert("Force settlement signal sent.");
    } catch (err) {
      console.error(err);
      alert(`Failed: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelCall = async () => {
    if (!activeCall || submitting) return;
    if (!window.confirm("Are you sure you want to cancel this capital call?")) return;
    setSubmitting(true);
    try {
      await postCancelCall(activeCall.id);
      alert("Cancel call signal sent.");
    } catch (err) {
      console.error(err);
      alert(`Failed: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  if (!activeCall) {
    return <div style={{ padding: '2rem', textAlign: 'center' }}>Loading capital call details...</div>;
  }

  const progress = activeCall.target > 0 ? Math.min((activeCall.received / activeCall.target) * 100, 100) : 0;
  const gap = Math.max(activeCall.target - activeCall.received, 0);
  const gapPct = activeCall.target > 0 ? ((gap / activeCall.target) * 100).toFixed(1) : '0.0';
  const isGap = gap > 0;
  
  // Bridge is only triggered when the workflow is in a finalized state (settled) AND there is a gap.
  const bridgeTriggered = activeCall.status === 'settled' && isGap;

  const getRiskLabel = (score: number) => {
    const s = score * 10;
    if (s >= 7) return 'High';
    if (s >= 4) return 'Med';
    return 'Low';
  };

  const getStatusBadge = (status: string) => {
    const s = status.toLowerCase();
    if (s === 'committed') return { label: 'Committed', cls: 'ccd-status-committed' };
    if (s === 'defaulted') return { label: 'Defaulted', cls: 'ccd-status-defaulted' };
    if (s === 'pending') return { label: 'Pending', cls: 'ccd-status-pending' };
    return { label: status || 'Pending', cls: 'ccd-status-pending' };
  };

  const deadline = new Date(activeCall.deadlineDate);
  const deadlineStr = deadline.toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: 'numeric' });

  return (
    <div className="ccd-page">
      {/* Header */}
      <div className="ccd-header-bar">
        <div className="ccd-header-left">
          <button className="ccd-back-btn" onClick={() => navigate('/capital-calls')}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
            Back
          </button>
          <div>
            <div className="ccd-title-row">
              <h1 className="ccd-call-id">{activeCall.id}</h1>
              <span className={`ccd-phase-badge ${activeCall.status === 'issued' ? 'phase-active' : activeCall.status === 'settled' ? 'phase-settled' : 'phase-pending'}`}>
                <span className="ccd-phase-dot"></span>
                Phase: {activeCall.status === 'issued' ? 'Collecting Responses (Active)' : activeCall.status === 'settled' ? 'Settled' : activeCall.status}
              </span>
            </div>
            <div className="ccd-subtitle">{activeCall.fund}</div>
            <div className="ccd-meta-row">
              <span>
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
                Deadline {deadlineStr}
              </span>
            </div>
          </div>
        </div>
      </div>

      <div className="ccd-body">
        {/* Left Column */}
        <div className="ccd-main-col">

          {/* Summary Cards */}
          <div className="ccd-summary-cards">
            <div className="ccd-summary-card">
              <div className="ccd-summary-label">Target</div>
              <div className="ccd-summary-value">{currencyFmt.format(activeCall.target)}</div>
            </div>
            <div className="ccd-summary-card">
              <div className="ccd-summary-label">Collected</div>
              <div className="ccd-summary-value" style={{ color: '#4f46e5' }}>{currencyFmt.format(activeCall.received)}</div>
            </div>
            <div className={`ccd-summary-card ${isGap ? 'ccd-summary-card-alert' : ''}`}>
              <div className="ccd-summary-label">Gap</div>
              <div className="ccd-summary-value" style={{ color: isGap ? '#dc2626' : '#16a34a' }}>
                {currencyFmt.format(gap)}
                <span className="ccd-summary-pct"> {gapPct}%</span>
              </div>
            </div>
            <div className="ccd-summary-card">
              <div className="ccd-summary-label">LP Count</div>
              <div className="ccd-summary-value">{activeCall.lpCompletion || '—'}</div>
            </div>
            <div className="ccd-summary-card">
              <div className="ccd-summary-label">Bridge Facility</div>
              <div className={`ccd-bridge-badge ${bridgeTriggered ? 'bridge-active' : 'bridge-inactive'}`}>
                <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
                {bridgeTriggered ? 'Active' : 'Inactive'}
              </div>
            </div>
          </div>

          {/* Progress Bar */}
          <div className="ccd-progress-section">
            <div className="ccd-progress-header">
              <span className="ccd-progress-title">OPERATIONAL LIQUIDITY PROGRESS</span>
              <span className="ccd-progress-pct" style={{ color: bridgeTriggered ? '#dc2626' : '#4f46e5' }}>
                {progress.toFixed(1)}% toward threshold
              </span>
            </div>
            <div className="ccd-progress-track">
              <div className="ccd-progress-fill" style={{ width: `${progress}%`, background: bridgeTriggered ? '#dc2626' : '#4f46e5' }}></div>
            </div>
            {bridgeTriggered && (
              <div className="ccd-bridge-alert">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                Bridge Facility Activated: Liquidity gap exceeded operational threshold.
              </div>
            )}
          </div>

          {portfolioRisk && (
            <div className="ccd-section-card" style={{ padding: '1.25rem', marginBottom: '1.25rem', borderLeft: '4px solid #f59e0b' }}>
              <div className="ccd-section-title" style={{ marginBottom: '1rem' }}>PORTFOLIO RISK SUMMARY</div>
              <div style={{ display: 'flex', gap: '3rem', fontSize: '0.875rem' }}>
                <div>
                  <span style={{ color: '#6b7280', display: 'block', marginBottom: '0.25rem', textTransform: 'uppercase', fontSize: '0.75rem', fontWeight: 600 }}>Concentration Score (HHI)</span>
                  <strong style={{ fontSize: '1.5rem', color: '#111827' }}>{portfolioRisk.concentrationScore}</strong>
                </div>
                <div>
                  <span style={{ color: '#6b7280', display: 'block', marginBottom: '0.25rem', textTransform: 'uppercase', fontSize: '0.75rem', fontWeight: 600 }}>Top Risky LPs</span>
                  <strong style={{ fontSize: '1.1rem', color: '#dc2626' }}>{portfolioRisk.topRiskyLPs?.join(', ') || 'None'}</strong>
                </div>
              </div>
            </div>
          )}

          {/* LP Contribution Ledger */}
          <div className="ccd-section-card">
            <div className="ccd-section-header">
              <span className="ccd-section-title">LP CONTRIBUTION LEDGER</span>
              <div className="ccd-section-actions-row">
                <button className="ccd-icon-btn">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>
                </button>
                <button className="ccd-icon-btn">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>
                </button>
              </div>
            </div>

            {lpLoading ? (
              <div className="ccd-empty-lp">
                <p style={{ color: '#6b7280' }}>Loading LP participation data…</p>
              </div>
            ) : callLPs.length === 0 ? (
              <div className="ccd-empty-lp">
                <p>No LP participation data yet. LPs will appear once the workflow registers their entries.</p>
              </div>
            ) : (
              <div className="ccd-table-wrap">
                <table className="ccd-table">
                  <thead>
                    <tr>
                      <th>LP Name</th>
                      <th>ID</th>
                      <th>Commitment</th>
                      <th>Requested</th>
                      <th>Status</th>
                      <th>Risk</th>
                      <th>Review</th>
                    </tr>
                  </thead>
                  <tbody>
                    {callLPs.map(lp => {
                      // Overlay risky LP data if available
                      const riskyData = riskyLPMap[lp.lpId] || null;
                      const reviewed = reviewedLPs[lp.lpId];

                      // Determine row display state
                      const isRisky = !!riskyData && !reviewed;
                      const isWaived = !!reviewed && reviewed === 'waived';
                      const isEnforced = !!reviewed && reviewed === 'enforced';

                      // Risk score: prefer risky overlay, fall back to DB value
                      const riskScore = riskyData ? riskyData.riskScore : lp.riskScore;
                      const isHighRisk = riskScore !== null && riskScore * 10 >= 7;
                      const riskNum = riskScore !== null ? (riskScore * 10).toFixed(1) : '—';
                      const riskLabel = riskScore !== null ? getRiskLabel(riskScore) : '—';

                      const badge = getStatusBadge(lp.status);

                      const rowClass = isRisky ? 'row-risky' : isWaived ? 'row-waived' : isEnforced ? 'row-enforced' : '';

                      return (
                        <tr key={lp.lpId} className={`ccd-lp-row ${rowClass}`}>
                          <td className="ccd-lp-name">{lp.lpId.replace(/-/g, ' ')}</td>
                          <td className={`ccd-lp-id ${isRisky ? 'lp-id-risky' : ''}`}>{lp.lpId}</td>
                          <td>{lp.commitmentUSD > 0 ? currencyFmt.format(lp.commitmentUSD) : '—'}</td>
                          <td>{lp.drawAmountUSD > 0 ? currencyFmt.format(lp.drawAmountUSD) : '—'}</td>
                          <td>
                            <span className={`ccd-status-pill ${badge.cls}`}>{badge.label}</span>
                          </td>
                          <td>
                            {riskScore !== null ? (
                              <span className={`ccd-risk-tag ${isHighRisk ? 'risk-high' : 'risk-low'}`}>
                                {riskNum} / {riskLabel}
                              </span>
                            ) : (
                              <span className="ccd-risk-tag risk-pending">—</span>
                            )}
                          </td>
                          <td>
                            {isRisky && riskyData && (
                              <button
                                className="ccd-flagged-btn"
                                onClick={() => setSelectedLP(riskyData)}
                              >
                                Flagged
                              </button>
                            )}
                            {isWaived && (
                              <span className="ccd-review-approved">Waived</span>
                            )}
                            {isEnforced && (
                              <span className="ccd-review-enforced">Enforced</span>
                            )}
                            {!isRisky && !isWaived && !isEnforced && (
                              <span className="ccd-review-neutral">—</span>
                            )}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Bottom row: Workflow Controls + Audit */}
          <div className="ccd-bottom-row">
            <div className="ccd-section-card ccd-workflow-controls">
              <div className="ccd-section-title" style={{ marginBottom: '1rem' }}>WORKFLOW CONTROLS</div>
              <div className="ccd-workflow-btns">
                <button className="ccd-ctrl-btn ccd-ctrl-force" onClick={handleForceSettlement} disabled={submitting || activeCall.status === 'settled' || activeCall.status === 'cancelled'}>Force Settlement</button>
                <button className="ccd-ctrl-btn ccd-ctrl-cancel" onClick={handleCancelCall} disabled={submitting || activeCall.status === 'settled' || activeCall.status === 'cancelled'}>Cancel Call</button>
              </div>
            </div>

            <div className="ccd-section-card ccd-audit-section">
              <div className="ccd-section-title" style={{ marginBottom: '1rem' }}>AUDIT & REPORTING</div>
              <div className="ccd-audit-btns">
                {activeCall.status === 'settled' ? (
                  <button className="ccd-audit-btn" onClick={() => navigate(`/reports/${activeCall.id}`)}>
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                    PDF
                  </button>
                ) : (
                  <button className="ccd-audit-btn ccd-audit-btn-disabled" disabled>
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                    PDF
                  </button>
                )}
                <button
                  className="ccd-audit-btn"
                  onClick={() => window.open(`/reports/${activeCall.id}.json`, '_blank')}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
                  JSON
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Right: Review Panel OR Timeline */}
        <div className="ccd-right-col">
          {selectedLP ? (
            <div className="ccd-section-card ccd-review-panel">
              <div className="ccd-review-panel-header">
                <span className="ccd-review-critical-label">
                  CRITICAL REVIEW REQUIRED
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#dc2626" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                </span>
                <button className="ccd-panel-close" onClick={() => setSelectedLP(null)}>×</button>
              </div>
              <div className="ccd-review-entity-row">
                <span>Entity</span>
                <strong>{selectedLP.lpId}</strong>
              </div>
              <div className="ccd-review-entity-row">
                <span>Identifier</span>
                <strong>{selectedLP.lpId}</strong>
              </div>
              <div className="ccd-review-risk-score">
                <span className="ccd-review-score-num">{(selectedLP.riskScore * 10).toFixed(1)}</span>
                <span className="ccd-review-score-denom">/10</span>
              </div>
              <div className="ccd-review-warning-box">
                High probability of continued non-payment. Entity has missed the 24-hour liquidity window twice.
              </div>
              <div className="ccd-lpr-modal-details">
                <div className="lpr-detail-row">
                  <span>Call ID</span>
                  <strong>{selectedLP.callId}</strong>
                </div>
                <div className="lpr-detail-row">
                  <span>Risk Score</span>
                  <strong style={{ color: '#dc2626' }}>{(selectedLP.riskScore * 10).toFixed(1)} / 10</strong>
                </div>
                <div className="lpr-detail-row">
                  <span>Total Commitment</span>
                  <strong>{currencyFmt.format(selectedLP.commitmentUSD)}</strong>
                </div>
                <div className="lpr-detail-row">
                  <span>Amount Due (Pro-Rata)</span>
                  <strong>{currencyFmt.format(selectedLP.drawAmountUSD)}</strong>
                </div>
              </div>
              <div className="ccd-review-actions">
                <button className="lpr-action-waive" onClick={() => handleAction('waive')} disabled={submitting}>
                  Waive Warning
                </button>
                <button className="lpr-action-enforce" onClick={() => handleAction('enforce')} disabled={submitting}>
                  Enforce Warning
                </button>
              </div>
            </div>
          ) : (
          <div className="ccd-section-card ccd-timeline-panel">
              <div className="ccd-section-title">OPERATION TIMELINE</div>
              <div className="ccd-timeline">
                {timelineEvents.length === 0 ? (
                  <div style={{ color: '#6b7280', fontSize: '0.85rem', padding: '1rem 0' }}>
                    No workflow events recorded yet. Timeline updates as activities complete.
                  </div>
                ) : (
                  timelineEvents.map((ev, idx) => (
                    <div key={idx} className="ccd-timeline-item">
                      <div className="ccd-timeline-time">
                        {ev.timestamp
                          ? new Date(ev.timestamp).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
                          : '—'}
                      </div>
                      <div className="ccd-timeline-indicator">
                        <div className="ccd-timeline-dot" style={{ background: ev.color }}></div>
                        {idx < timelineEvents.length - 1 && <div className="ccd-timeline-line"></div>}
                      </div>
                      <div className="ccd-timeline-content">
                        <div className="ccd-timeline-title">{ev.title}</div>
                        <div className="ccd-timeline-desc">{ev.name}</div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
