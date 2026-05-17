import { useState, useEffect } from 'react';
import '../styles/LoginPage.css';

interface LoginPageProps {
  onLogin: () => void;
  onLPLogin: (lpId: string) => void;
}

interface LP {
  lpId: string;
  commitmentUSD: number;
  email: string;
}

function LoginPage({ onLogin, onLPLogin }: LoginPageProps) {
  const [lps, setLPs] = useState<LP[]>([]);
  const [selectedLP, setSelectedLP] = useState('');
  const [lpError, setLPError] = useState('');

  useEffect(() => {
    fetch('/api/lps')
      .then(r => r.json())
      .then((data: LP[]) => setLPs(Array.isArray(data) ? data : []))
      .catch(() => setLPs([]));
  }, []);

  const handleLPLaunch = () => {
    if (!selectedLP) {
      setLPError('Please select an LP to continue.');
      return;
    }
    setLPError('');
    onLPLogin(selectedLP);
  };

  return (
    <div className="login-container">
      <div className="login-header">
        <div className="login-logo-container">
          <svg className="login-bank-icon" xmlns="http://www.w3.org/2000/svg" width="28" height="28"
            viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
            strokeLinecap="round" strokeLinejoin="round">
            <path d="M3 21h18"></path>
            <path d="M3 18h18"></path>
            <path d="M5 18v-9"></path>
            <path d="M9 18v-9"></path>
            <path d="M15 18v-9"></path>
            <path d="M19 18v-9"></path>
            <path d="M2 9l10-5 10 5"></path>
          </svg>
        </div>
        <h1>Intelligent Capital Call &amp; Liquidity Orchestration</h1>
        <p className="login-subtitle">Durable orchestration for capital call operations</p>
      </div>

      <div className="login-cards-container">
        {/* GP Card */}
        <div className="login-card gp-card">
          <div className="login-card-icon-container">
            <svg className="login-card-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24"
              viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
              strokeLinecap="round" strokeLinejoin="round">
              <rect x="4" y="2" width="16" height="20" rx="2" ry="2"></rect>
              <path d="M9 22v-4h6v4"></path>
              <path d="M8 6h.01"></path>
              <path d="M16 6h.01"></path>
              <path d="M12 6h.01"></path>
              <path d="M12 10h.01"></path>
              <path d="M12 14h.01"></path>
              <path d="M16 10h.01"></path>
              <path d="M16 14h.01"></path>
              <path d="M8 10h.01"></path>
              <path d="M8 14h.01"></path>
            </svg>
          </div>
          <h2>Enter as GP</h2>
          <p>Access your GP dashboard to manage funds, trigger capital calls, and monitor LP commitments in real-time.</p>
          <div className="login-spacer"></div>
          <button className="login-primary-button" onClick={onLogin}>
            Continue to GP Portal
            <svg className="login-arrow-icon" xmlns="http://www.w3.org/2000/svg" width="18" height="18"
              viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
              strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14"></path>
              <path d="m12 5 7 7-7 7"></path>
            </svg>
          </button>
        </div>

        {/* LP Card */}
        <div className="login-card lp-card">
          <div className="login-card-icon-container">
            <svg className="login-card-icon" xmlns="http://www.w3.org/2000/svg" width="24" height="24"
              viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
              strokeLinecap="round" strokeLinejoin="round">
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
              <circle cx="9" cy="7" r="4"></circle>
              <circle cx="19" cy="11" r="3"></circle>
              <path d="m22 14-2.1-2.1"></path>
            </svg>
          </div>
          <h2>Enter as LP</h2>
          <p>View your portfolio exposure, review pending capital calls, and manage your institutional profile.</p>
          <div className="login-spacer"></div>

          <div className="login-input-group">
            <label htmlFor="lp-select">Select LP Identity</label>
            <div className="login-select-wrapper">
              <select
                id="lp-select"
                className="login-lp-select"
                value={selectedLP}
                onChange={e => { setSelectedLP(e.target.value); setLPError(''); }}
              >
                <option value="" disabled>Select Institutional LP…</option>
                {lps.length === 0 && (
                  <option disabled>Loading…</option>
                )}
                {lps.map(lp => (
                  <option key={lp.lpId} value={lp.lpId}>
                    {lp.lpId.toUpperCase()}
                  </option>
                ))}
              </select>
              <svg className="login-chevron-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16"
                viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
                strokeLinecap="round" strokeLinejoin="round">
                <path d="m6 9 6 6 6-6"></path>
              </svg>
            </div>
            {lpError && (
              <span style={{ fontSize: '0.8rem', color: '#ef4444', marginTop: '0.375rem', display: 'block' }}>
                {lpError}
              </span>
            )}
          </div>

          <button className="login-secondary-button" onClick={handleLPLaunch}>
            Launch LP Console
            <svg className="login-enter-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16"
              viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
              strokeLinecap="round" strokeLinejoin="round">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
              <polyline points="15 3 21 3 21 9"></polyline>
              <line x1="10" y1="14" x2="21" y2="3"></line>
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}

export default LoginPage;