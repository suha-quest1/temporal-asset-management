import { useState, useEffect } from 'react';
import '../styles/LPRiskTrackerPage.css';
import { getRiskyLPs, postGPDecision } from '../api/CapitalCall';

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

const LPRiskTrackerPage = () => {
  const [riskyLPs, setRiskyLPs] = useState<RiskyLP[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedLP, setSelectedLP] = useState<RiskyLP | null>(null);

  useEffect(() => {
    const fetchRiskyLPs = async () => {
      try {
        const data = await getRiskyLPs();
        setRiskyLPs(data);
        setLoading(false);
      } catch (err) {
        console.error('Failed to fetch risky LPs', err);
        setLoading(false);
      }
    };

    fetchRiskyLPs();
    const interval = setInterval(() => { fetchRiskyLPs(); }, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleReviewClick = (lp: RiskyLP) => {
    setSelectedLP(lp);
  };

  const closeModal = () => {
    setSelectedLP(null);
  };

  const handleAction = async (action: 'waive' | 'enforce') => {
    if (!selectedLP) return;
    try {
      await postGPDecision(selectedLP.callId, {
        lpId: selectedLP.lpId,
        action: action,
        gpName: 'GP Admin'
      });
      // Close modal on success, table will refresh via polling
      closeModal();
    } catch (err) {
      console.error(`Failed to ${action} LP`, err);
      alert(`Failed to execute action: ${err}`);
    }
  };

  const currencyFmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });

  return (
    <div className="lpr-page cc-page">
      <nav className="cc-navbar">
        <div className="cc-nav-brand">Intelligent Capital Call & Liquidity Orchestration</div>
      </nav>

      <main className="lpr-main-content cc-main-content">
        <header className="lpr-header cc-header">
          <div>
            <h1 className="cc-page-title">LP Risk Tracker</h1>
            <p className="cc-page-subtitle">Monitor and resolve at-risk LP commitments in active workflows.</p>
          </div>
        </header>

        <div className="cc-table-container">
          <div className="cc-table-wrapper">
            <table className="cc-table">
              <thead>
                <tr>
                  <th>LP ID</th>
                  <th>RISK SCORE</th>
                  <th>ALERT TYPE</th>
                  <th>FLAGGED</th>
                  <th>AT-RISK AMOUNT</th>
                  <th>ACTION</th>
                </tr>
              </thead>
              <tbody>
                {loading && riskyLPs.length === 0 ? (
                  <tr><td colSpan={6} style={{textAlign: 'center', padding: '2rem'}}>Loading...</td></tr>
                ) : riskyLPs.length === 0 ? (
                  <tr><td colSpan={6} style={{textAlign: 'center', padding: '2rem'}}>No high-risk LPs detected at this time.</td></tr>
                ) : (
                  riskyLPs.map((lp, idx) => (
                    <tr key={`${lp.callId}-${lp.lpId}-${idx}`}>
                      <td>
                        <div className="cc-cell-title">{lp.lpId}</div>
                        <div className="cc-cell-sub">{lp.callId}</div>
                      </td>
                      <td>
                        <div className="lpr-risk-badge cc-badge" style={{ backgroundColor: `rgba(220, 38, 38, ${lp.riskScore})`, color: 'white' }}>
                          {(lp.riskScore * 10).toFixed(1)}/10
                        </div>
                      </td>
                      <td>
                        <span style={{ color: '#dc2626', fontWeight: 600 }}>High Default Probability</span>
                      </td>
                      <td>
                        {new Date(lp.flaggedAt).toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: 'numeric' })}
                      </td>
                      <td className="cc-target-amount" style={{ color: '#f59e0b' }}>
                        {currencyFmt.format(lp.drawAmountUSD)}
                      </td>
                      <td className="cc-action-cell">
                        <button className="lpr-review-btn" onClick={() => handleReviewClick(lp)}>Review</button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      </main>

      {/* GP Decision Modal */}
      {selectedLP && (
        <div className="cc-modal-overlay">
          <div className="cc-modal-content">
            <div className="cc-modal-header">
              <h2>GP Review: High-Risk LP — {selectedLP.lpId}</h2>
              <button className="cc-modal-close" onClick={closeModal}>×</button>
            </div>
            <div className="cc-modal-body">
              <div className="lpr-modal-warning">
                <strong>ML Risk Alert:</strong> This LP has been flagged with a high probability of default
                by the predictive ML pipeline. Review the details below and choose a governance action.
              </div>
              <div className="lpr-modal-details">
                <div className="lpr-detail-row">
                  <span>Call ID</span>
                  <strong>{selectedLP.callId}</strong>
                </div>
                <div className="lpr-detail-row">
                  <span>Risk Score</span>
                  <strong style={{ color: '#dc2626' }}>{(selectedLP.riskScore * 10).toFixed(1)} / 10</strong>
                </div>
                <div className="lpr-detail-row">
                  <span>LP Status</span>
                  <strong>{selectedLP.lpStatus.toUpperCase()}</strong>
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
            </div>
            <div className="cc-modal-footer">
              {/* Waive: accept the risky contribution, workflow continues normally */}
              <button className="lpr-action-waive" onClick={() => handleAction('waive')}>
                Waive Warning
              </button>
              {/* Enforce: send compliance email warning; contribution remains intact */}
              <button className="lpr-action-enforce" onClick={() => handleAction('enforce')}>
                Enforce Warning
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default LPRiskTrackerPage;
