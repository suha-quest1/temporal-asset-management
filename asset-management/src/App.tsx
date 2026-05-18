import { useState } from 'react'
import './App.css'
import CapitalCallPage from './pages/CapitalCallPage'
import LoginPage from './pages/LoginPage'
import LimitedPartnerPage from './pages/LimitedPartnerPage'
import LPRiskTrackerPage from './pages/LPRiskTrackerPage'
import ReportViewerPage from './pages/ReportViewerPage'
import './styles/GPPortalLayout.css'

type Page = 'login' | 'gp-calls' | 'gp-lps' | 'lp' | 'report'

function App() {
  const [page, setPage] = useState<Page>('login')
  const [activeLPId, setActiveLPId] = useState<string>('')
  const [activeReportCallId, setActiveReportCallId] = useState<string>('')

  if (page.startsWith('gp-')) {
    const isCallsView = page === 'gp-calls';
    const isLpsView = page === 'gp-lps';

    return (
      <div className="gp-layout">
        {/* Left Sidebar Navigation */}
        <aside className="gp-sidebar">
          <nav className="gp-sidebar-nav">
            <button 
              className={`gp-sidebar-item ${isCallsView ? 'active' : ''}`}
              onClick={() => setPage('gp-calls')}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>
              Capital Calls
            </button>
            <button 
              className={`gp-sidebar-item ${isLpsView ? 'active' : ''}`}
              onClick={() => setPage('gp-lps')}
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
              LPs (Risk Tracker)
            </button>
          </nav>
          
          <div className="gp-sidebar-footer">
            <button onClick={() => setPage('login')}>
              Log Out (Back to Login)
            </button>
          </div>
        </aside>

        {/* Main Content Area */}
        <main className="gp-content">
          {isCallsView && <CapitalCallPage onViewReport={(callId) => { setActiveReportCallId(callId); setPage('report'); }} />}
          {isLpsView && <LPRiskTrackerPage />}
        </main>
      </div>
    )
  }

  if (page === 'report' && activeReportCallId) {
    return (
      <ReportViewerPage
        callId={activeReportCallId}
        onBack={() => { setActiveReportCallId(''); setPage('gp-calls'); }}
      />
    )
  }

  if (page === 'lp' && activeLPId) {
    return (
      <LimitedPartnerPage
        lpId={activeLPId}
        onBack={() => { setActiveLPId(''); setPage('login'); }}
      />
    )
  }

  return (
    <LoginPage
      onLogin={() => setPage('gp-calls')}
      onLPLogin={(lpId) => { setActiveLPId(lpId); setPage('lp'); }}
    />
  )
}

export default App
