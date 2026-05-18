import { useState, useEffect } from 'react';
import '../styles/CapitalCallPage.css';
import StartCapitalCallModal from '../components/StartCapitalCallModal';
import CapitalCallRow from '../components/CapitalCallRow';
import type { CapitalCall } from '../components/CapitalCallRow';
import { getCapitalCalls } from '../api/CapitalCall';

interface CapitalCallPageProps {
  onViewReport?: (callId: string) => void;
}

const CapitalCallPage = ({ onViewReport }: CapitalCallPageProps) => {
  const [activeTab, setActiveTab] = useState('All Calls');
  // State for controlling the visibility of the "Start Capital Call" modal
  const [isModalOpen, setIsModalOpen] = useState(false);

  interface DashboardStats {
    totalCalledYTD: string;
    pendingLiquidity: string;
    avgLPResponse: string;
    activeCalls: string;
  }

  const [stats, setStats] = useState<DashboardStats>({
    totalCalledYTD: '—',
    pendingLiquidity: '—',
    avgLPResponse: '—',
    activeCalls: '—',
  });

  const [calls, setCalls] = useState<CapitalCall[]>([]);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const data = await fetch('/api/dashboard/stats').then(r => r.json());
        setStats(data);
      } catch (err) {
        console.error('Failed to fetch dashboard stats', err);
      }
    };

    const fetchCalls = async () => {
      try {
        const data = await getCapitalCalls();
        if (Array.isArray(data)) {
          setCalls(data);
        }
      } catch (err) {
        console.error('Failed to fetch capital calls', err);
      }
    };

    fetchStats();
    fetchCalls();
    const interval = setInterval(() => { fetchStats(); fetchCalls(); }, 5000);
    return () => clearInterval(interval);
  }, []);

  const statCards = [
    { label: 'TOTAL CALLED (YTD)', value: stats.totalCalledYTD },
    { label: 'PENDING LIQUIDITY', value: stats.pendingLiquidity, valueColor: '#4f46e5' },
    { label: 'AVG. LP RESPONSE', value: stats.avgLPResponse },
    { label: 'ACTIVE CALLS', value: stats.activeCalls },
  ];

  return (
    <div className="cc-page">
      {/* Top Navbar */}
      <nav className="cc-navbar">
        <div className="cc-nav-brand">Intelligent Capital Call & Liquidity Orchestration</div>
        <div className="cc-search-container">
          <svg className="cc-search-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
          <input type="text" placeholder="Search capital calls..." className="cc-search-input" />
        </div>
      </nav>

      <main className="cc-main-content">
        {/* Header Section */}
        <header className="cc-header">
          <div>
            <h1 className="cc-page-title">Capital Calls</h1>
            <p className="cc-page-subtitle">Manage and track liquidity deployments across your portfolio funds.</p>
          </div>
          <button className="cc-new-call-btn" onClick={() => setIsModalOpen(true)}>
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="16"></line><line x1="8" y1="12" x2="16" y2="12"></line></svg>
            Start New Capital Call
          </button>
        </header>

        {/* Stats Section */}
        <div className="cc-stats-grid">
          {statCards.map((stat, idx) => (
            <div key={idx} className="cc-stat-card">
              <div className="cc-stat-label">{stat.label}</div>
              <div className="cc-stat-value" style={stat.valueColor ? { color: stat.valueColor } : {}}>{stat.value}</div>
            </div>
          ))}
        </div>

        {/* Table Section */}
        <div className="cc-table-container">
          {/* Tabs */}
          <div className="cc-table-tabs">
            {['All Calls', 'Active', 'Pending'].map(tab => (
              <button 
                key={tab} 
                className={`cc-tab ${activeTab === tab ? 'active' : ''}`}
                onClick={() => setActiveTab(tab)}
              >
                {tab}
              </button>
            ))}
          </div>

          {/* Table */}
          <div className="cc-table-wrapper">
            <table className="cc-table">
              <thead>
                <tr>
                  <th>CALL ID</th>
                  <th>TARGET AMOUNT</th>
                  <th>RECEIVED</th>
                  <th>LP COMPLETION</th>
                  <th>DEADLINE</th>
                  <th>STATUS</th>
                  <th>ACTION</th>
                </tr>
              </thead>
              <tbody>
                {calls.map((call) => (
                  <CapitalCallRow key={call.id} call={call} onViewReport={onViewReport} />
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          <div className="cc-pagination">
            <div className="cc-pagination-info">
              Showing 1-4 of 12 Capital Calls
            </div>
            <div className="cc-pagination-controls">
              <button className="cc-page-btn cc-page-arrow">&lt;</button>
              <button className="cc-page-btn active">1</button>
              <button className="cc-page-btn">2</button>
              <button className="cc-page-btn">3</button>
              <button className="cc-page-btn cc-page-arrow">&gt;</button>
            </div>
          </div>
        </div>
      </main>

      {/* Conditionally render the StartCapitalCallModal overlay */}
      {isModalOpen && (
        <StartCapitalCallModal 
          isOpen={isModalOpen} 
          onClose={() => setIsModalOpen(false)} 
        />
      )}
    </div>
  );
};

export default CapitalCallPage;
