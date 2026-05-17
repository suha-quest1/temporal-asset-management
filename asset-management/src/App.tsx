import { useState } from 'react'
import './App.css'
import CapitalCallPage from './pages/CapitalCallPage'
import LoginPage from './pages/LoginPage'
import LimitedPartnerPage from './pages/LimitedPartnerPage'

type Page = 'login' | 'gp' | 'lp'

function App() {
  const [page, setPage] = useState<Page>('login')
  const [activeLPId, setActiveLPId] = useState<string>('')

  if (page === 'gp') {
    return (
      <div style={{ position: 'relative' }}>
        <CapitalCallPage />
        <button
          onClick={() => setPage('login')}
          style={{
            position: 'fixed', bottom: '1rem', right: '1rem', zIndex: 9999,
            padding: '0.5rem 1rem', background: '#111827', color: 'white',
            border: 'none', borderRadius: '8px', cursor: 'pointer',
            boxShadow: '0 4px 6px rgba(0,0,0,0.1)', fontSize: '0.875rem',
          }}
        >
          Back to Login
        </button>
      </div>
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
      onLogin={() => setPage('gp')}
      onLPLogin={(lpId) => { setActiveLPId(lpId); setPage('lp'); }}
    />
  )
}

export default App
