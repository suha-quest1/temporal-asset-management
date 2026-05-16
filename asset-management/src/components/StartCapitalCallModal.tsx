import type { FC } from 'react';
import './StartCapitalCallModal.css';

interface StartCapitalCallModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const dummyLPs = [
  {
    id: '#LP-2940',
    name: 'Blue Chip Pension Fund',
    committed: '$50,000,000.00',
    lastCall: '12d ago',
  },
  {
    id: '#LP-8812',
    name: 'Evergreen Global Endowment',
    committed: '$25,000,000.00',
    lastCall: '45d ago',
  },
  {
    id: '#LP-1109',
    name: 'Sovereign Wealth Horizon',
    committed: '$120,000,000.00',
    lastCall: '3m ago',
  },
];

const StartCapitalCallModal: FC<StartCapitalCallModalProps> = ({ isOpen, onClose }) => {
  // 1. Conditional rendering: if the modal is not open, return null to render nothing.
  if (!isOpen) return null;

  // 2. Event handler: logs dummy payload and closes the modal.
  const handleStart = () => {
    console.log('Initiating Capital Call with dummy payload...', {
      targetAmount: 0,
      deadlineDays: 10,
      selectedPartners: dummyLPs.map(lp => lp.id)
    });
    onClose();
  };

  // 3. Overlay wrapping: clicking the overlay backdrop triggers onClose()
  //    clicking inside the content stops event propagation so it doesn't close.
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
                <input type="text" placeholder="0.00" defaultValue="0.00" />
              </div>
            </div>
            <div className="cc-modal-field">
              <label>Deadline Days</label>
              <div className="cc-input-wrapper">
                <input type="number" defaultValue={10} />
                <span className="cc-input-suffix">Days</span>
              </div>
            </div>
          </div>

          {/* Institutional Partner Selection Section */}
          <div className="cc-modal-partners-section">
            <div className="cc-modal-partners-header">
              <label>Select Institutional Partners</label>
              <button className="cc-select-all-btn">Select All</button>
            </div>

            {/* Search Bar */}
            <div className="cc-modal-search">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <input type="text" placeholder="Search partners by name or ID..." />
            </div>

            {/* Scrollable Partner List */}
            <div className="cc-modal-partner-list">
              {dummyLPs.map((lp, idx) => (
                <div key={idx} className="cc-partner-item">
                  <div className="cc-partner-checkbox">
                    <input type="checkbox" />
                  </div>
                  <div className="cc-partner-details">
                    <div className="cc-partner-name">{lp.name}</div>
                    <div className="cc-partner-meta">Committed: {lp.committed} • Last Call: {lp.lastCall}</div>
                  </div>
                  <div className="cc-partner-id">ID: {lp.id}</div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Footer Section */}
        <div className="cc-modal-footer">
          <button className="cc-modal-cancel-btn" onClick={onClose}>Cancel</button>
          <button className="cc-modal-start-btn" onClick={handleStart}>Start Capital Call</button>
        </div>
      </div>
    </div>
  );
};

export default StartCapitalCallModal;
