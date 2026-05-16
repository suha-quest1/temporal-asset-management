import { useState } from 'react'
import './App.css'
import CapitalCallPage from './pages/CapitalCallPage'
import LoginPage from './pages/LoginPage'

function App() {
  const [currentPage, setCurrentPage] = useState<'login' | 'cc'>('login')

  if (currentPage === 'cc') {
    return (
      <div style={{ position: 'relative' }}>
        <CapitalCallPage />
        <button 
          onClick={() => setCurrentPage('login')} 
          style={{ position: 'fixed', bottom: '1rem', right: '1rem', zIndex: 9999, padding: '0.5rem 1rem', background: '#111827', color: 'white', border: 'none', borderRadius: '8px', cursor: 'pointer', boxShadow: '0 4px 6px rgba(0,0,0,0.1)' }}>
          Back to Login
        </button>
      </div>
    )
  }

  return <LoginPage onLogin={() => setCurrentPage('cc')} />
}

export default App
