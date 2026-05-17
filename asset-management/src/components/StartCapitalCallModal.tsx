import './StartCapitalCallModal.css';
import { useState, useEffect, type FC } from 'react';
import { startCapitalCall } from '../api/CapitalCall';

interface LP {
  lpId: string;
  commitmentUSD: number;
  email: string;
}

interface StartCapitalCallModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const StartCapitalCallModal: FC<StartCapitalCallModalProps> = ({ isOpen, onClose }) => {
  const [targetAmount, setTargetAmount] = useState('');
  const [deadlineDays, setDeadlineDays] = useState(10);
  const [selectedPartners, setSelectedPartners] = useState<string[]>([]);
  const [lps, setLPs] = useState<LP[]>([]);
  const [loadingLPs, setLoadingLPs] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  // Fetch LPs from backend when modal opens
  useEffect(() => {
    if (!isOpen) return;

    setLoadingLPs(true);
    fetch('/api/lps')
      .then(r => r.json())
      .then((data: LP[]) => {
        setLPs(Array.isArray(data) ? data : []);
      })
      .catch(err => {
        console.error('Failed to load LPs:', err);
        setLPs([]);
      })
      .finally(() => setLoadingLPs(false));
  }, [isOpen]);

  if (!isOpen) return null;

  const handleSelectAll = () => {
    if (selectedPartners.length === lps.length) {
      setSelectedPartners([]);
    } else {
      setSelectedPartners(lps.map(lp => lp.lpId));
    }
  };

  const handleStart = async () => {
    if (!targetAmount || Number(targetAmount) <= 0) {
      setError('Please enter a valid target amount.');
      return;
    }
    if (selectedPartners.length === 0) {
      setError('Please select at least one LP.');
      return;
    }
    setError('');
    setSubmitting(true);

    try {
      // Build LP list using data from the backend — no frontend-owned commitments
      const lpList = selectedPartners.map(id => {
        const lp = lps.find(l => l.lpId === id)!;
        return {
          lpId: lp.lpId,
          commitmentUSD: lp.commitmentUSD,
          email: lp.email,
        };
      });

      // Backend generates callId — frontend only sends the business inputs
      const payload = {
        fundId: 'fund-1',
        targetAmountUSD: Number(targetAmount),
        deadlineDays: deadlineDays,
        lpList: lpList,
      };

      console.log('Sending payload:', payload);
      const result = await startCapitalCall(payload);
      console.log('Workflow started:', result);

      // Reset form
      setTargetAmount('');
      setDeadlineDays(10);
      setSelectedPartners([]);
      onClose();
    } catch (err) {
      console.error(err);
      setError('Failed to start capital call. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="cc-modal-overlay" onClick={onClose}>
      <div className="cc-modal-content" onClick={(e) => e.stopPropagation()}>
        {/* Header Section */}
        <div className="cc-modal-header">
          <div className="cc-modal-title-group">
            <div className="cc-modal-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"></path>
                <path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"></path>
                <path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"></path>
                <path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"></path>
              </svg>
            </div>
            <h2>Initiate New Capital Call</h2>
          </div>
          <button className="cc-modal-close" onClick={onClose}>
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        {/* Body Section */}
        <div className="cc-modal-body">
          {/* Top Form Row */}
          <div className="cc-modal-row">
            <div className="cc-modal-field">
              <label>Target Amount (USD)</label>
              <div className="cc-input-wrapper">
                <span className="cc-input-prefix">$</span>
                <input
                  type="text"
                  placeholder="0.00"
                  value={targetAmount}
                  onChange={(e) => setTargetAmount(e.target.value)}
                />
              </div>
            </div>
            <div className="cc-modal-field">
              <label>Deadline Days</label>
              <div className="cc-input-wrapper">
                <input
                  type="number"
                  value={deadlineDays}
                  onChange={(e) => setDeadlineDays(Number(e.target.value))}
                />
                <span className="cc-input-suffix">Days</span>
              </div>
            </div>
          </div>

          {/* Institutional Partner Selection Section */}
          <div className="cc-modal-partners-section">
            <div className="cc-modal-partners-header">
              <label>Select Institutional Partners ({selectedPartners.length} selected)</label>
              <button className="cc-select-all-btn" onClick={handleSelectAll}>
                {selectedPartners.length === lps.length && lps.length > 0 ? 'Deselect All' : 'Select All'}
              </button>
            </div>

            {/* Scrollable Partner List */}
            <div className="cc-modal-partner-list">
              {loadingLPs ? (
                <div style={{ padding: '16px', color: '#9ca3af', textAlign: 'center' }}>
                  Loading partners...
                </div>
              ) : lps.length === 0 ? (
                <div style={{ padding: '16px', color: '#9ca3af', textAlign: 'center' }}>
                  No LPs available.
                </div>
              ) : (
                lps.map((lp) => (
                  <div key={lp.lpId} className="cc-partner-item">
                    <div className="cc-partner-checkbox">
                      <input
                        type="checkbox"
                        checked={selectedPartners.includes(lp.lpId)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedPartners([...selectedPartners, lp.lpId]);
                          } else {
                            setSelectedPartners(selectedPartners.filter(id => id !== lp.lpId));
                          }
                        }}
                      />
                    </div>
                    <div className="cc-partner-details">
                      <div className="cc-partner-name">{lp.lpId}</div>
                      <div className="cc-partner-meta">
                        Committed: ${lp.commitmentUSD.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                      </div>
                    </div>
                    <div className="cc-partner-id">{lp.email}</div>
                  </div>
                ))
              )}
            </div>
          </div>

          {error && (
            <div style={{ color: '#ef4444', fontSize: '0.875rem', marginTop: '8px', padding: '0 4px' }}>
              {error}
            </div>
          )}
        </div>

        {/* Footer Section */}
        <div className="cc-modal-footer">
          <button className="cc-modal-cancel-btn" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button className="cc-modal-start-btn" onClick={handleStart} disabled={submitting}>
            {submitting ? 'Starting...' : 'Start Capital Call'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default StartCapitalCallModal;
