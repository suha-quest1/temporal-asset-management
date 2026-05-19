import { Routes, Route, Navigate } from 'react-router-dom'
import './App.css'
import CapitalCallPage from './pages/CapitalCallPage'
import LoginPage from './pages/LoginPage'
import LimitedPartnerPage from './pages/LimitedPartnerPage'
import ReportViewerPage from './pages/ReportViewerPage'
import CapitalCallDetailsPage from './pages/CapitalCallDetailsPage'
import './styles/GPPortalLayout.css'

function App() {
  return (
    <Routes>
      <Route path="/" element={<LoginPage />} />
      <Route path="/capital-calls" element={
        <div className="gp-layout">
          <main className="gp-content">
            <CapitalCallPage />
          </main>
        </div>
      } />
      <Route path="/capital-calls/:callId" element={<CapitalCallDetailsPage />} />
      <Route path="/reports/:callId" element={<ReportViewerPage />} />
      <Route path="/lp/:lpId" element={<LimitedPartnerPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
