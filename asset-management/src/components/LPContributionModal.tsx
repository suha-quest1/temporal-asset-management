import './LPContributionModal.css';
import { useState, type FC } from 'react';
import { postLPResponse } from '../api/CapitalCall';

interface LPContributionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  callId: string;
  lpId: string;
  fundName: string;
  targetAmount: number;
  drawAmountUSD: number;
  deadlineDate: string;
}

const LPContributionModal: FC<LPContributionModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  callId,
  lpId,
  fundName,
  targetAmount,
  drawAmountUSD,
  deadlineDate,
}) => {
  const [amount, setAmount] = useState<string>(drawAmountUSD.toString());
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen) return null;

  const numericAmount = Number(amount);
  const isLessThanRequested = !isNaN(numericAmount) && numericAmount < drawAmountUSD;

  const handleSubmit = async () => {
    if (isNaN(numericAmount) || numericAmount <= 0) {
      setError('Please enter a valid contribution amount.');
      return;
    }

    setError('');
    setSubmitting(true);

    try {
      await postLPResponse(callId, {
        lpId: lpId,
        amount: numericAmount,
      });

      setSubmitted(true);
      
      // Briefly show "Submitted" then close
      setTimeout(() => {
        onSuccess();
      }, 1500);

    } catch (err: any) {
      console.error('Failed to post LP response:', err);
      setError(err.message || 'Failed to submit contribution. Please try again.');
      setSubmitting(false);
    }
  };

  const currencyFmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' });
  
  // Calculate remaining days for display
  const deadline = new Date(deadlineDate);
  const diffDays = Math.ceil((deadline.getTime() - Date.now()) / (1000 * 3600 * 24));
  const remainingText = diffDays > 0 ? `${diffDays} Days Remaining` : 'Past Due';

  return (
    <div className="cc-modal-overlay" onClick={onClose}>
      <div className="cc-modal-content lp-contrib-modal" onClick={(e) => e.stopPropagation()}>
        {/* Header Section */}
        <div className="cc-modal-header">
          <div className="cc-modal-title-group">
            <div className="cc-modal-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="12" y1="1" x2="12" y2="23"></line>
                <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
              </svg>
            </div>
            <h2>Respond to Capital Call</h2>
          </div>
          <button className="cc-modal-close" onClick={onClose} disabled={submitting || submitted}>
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        {/* Body Section */}
        <div className="cc-modal-body">
          {/* Summary Banner */}
          <div className="lp-contrib-summary">
            <div className="lp-contrib-summary-grid">
              <div className="lp-contrib-summary-item">
                <span className="lp-contrib-label">Capital Call</span>
                <span className="lp-contrib-value">{callId}</span>
              </div>
              <div className="lp-contrib-summary-item">
                <span className="lp-contrib-label">Fund</span>
                <span className="lp-contrib-value">{fundName}</span>
              </div>
              <div className="lp-contrib-summary-item">
                <span className="lp-contrib-label">Target Raise</span>
                <span className="lp-contrib-value">{currencyFmt.format(targetAmount)}</span>
              </div>
              <div className="lp-contrib-summary-item">
                <span className="lp-contrib-label">Deadline</span>
                <span className="lp-contrib-value" style={{ color: diffDays <= 0 ? '#dc2626' : 'inherit' }}>
                  {remainingText}
                </span>
              </div>
            </div>
            <div className="lp-contrib-highlight-row">
              <span className="lp-contrib-label">Your Required Share:</span>
              <span className="lp-contrib-highlight-value">{currencyFmt.format(drawAmountUSD)}</span>
            </div>
          </div>

          {/* Payment Input Section */}
          <div className="cc-modal-field lp-contrib-input-field">
            <label>Contribution Amount (USD)</label>
            <div className="cc-input-wrapper">
              <span className="cc-input-prefix">$</span>
              <input
                type="number"
                min="0"
                step="0.01"
                placeholder="0.00"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                disabled={submitting || submitted}
              />
            </div>
            {isLessThanRequested && (
              <div className="lp-contrib-warning">
                Amount is less than requested contribution.
              </div>
            )}
          </div>

          {error && (
            <div style={{ color: '#ef4444', fontSize: '0.875rem', marginTop: '12px', padding: '0 4px' }}>
              {error}
            </div>
          )}
        </div>

        {/* Footer Section */}
        <div className="cc-modal-footer">
          <button 
            className="cc-modal-cancel-btn" 
            onClick={onClose} 
            disabled={submitting || submitted}
          >
            Cancel
          </button>
          <button 
            className="cc-modal-start-btn" 
            onClick={handleSubmit} 
            disabled={submitting || submitted}
            style={submitted ? { backgroundColor: '#16a34a' } : {}}
          >
            {submitted ? 'Submitted' : submitting ? 'Processing...' : 'Submit Contribution'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default LPContributionModal;
