import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import '../styles/CapitalCallPage.css';
import StartCapitalCallModal from '../components/StartCapitalCallModal';
import CapitalCallRow from '../components/CapitalCallRow';
import type { CapitalCall } from '../components/CapitalCallRow';
import { getCapitalCalls } from '../api/CapitalCall';

const CapitalCallPage = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState('All Calls');
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 15;

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [stats, setStats] = useState({
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

  // 1. Filter
  const filteredCalls = calls.filter(call => {
    if (activeTab === 'Active') return call.status === 'issued';
    if (activeTab === 'Pending') return call.status === 'pending';
    if (activeTab === 'Settled') return call.status === 'settled';
    return true;
  });

  // 2. Search
  const searchedCalls = filteredCalls.filter(call => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return call.id.toLowerCase().includes(q) || 
           call.fund.toLowerCase().includes(q) || 
           call.status.toLowerCase().includes(q);
  });

  // 3. Paginate
  const totalPages = Math.ceil(searchedCalls.length / itemsPerPage) || 1;
  // Ensure current page is valid when filtering changes
  const validPage = Math.min(currentPage, totalPages);
  
  const startIndex = (validPage - 1) * itemsPerPage;
  const paginatedCalls = searchedCalls.slice(startIndex, startIndex + itemsPerPage);

  const statCards = [
    { label: 'TOTAL CALLED (YTD)', value: stats.totalCalledYTD },
    { label: 'PENDING LIQUIDITY', value: stats.pendingLiquidity, valueColor: '#4f46e5' },
    { label: 'AVG. LP RESPONSE', value: stats.avgLPResponse },
    { label: 'ACTIVE CALLS', value: stats.activeCalls },
  ];

  return (
    <div className="cc-page">
      <nav className="cc-navbar">
        <div className="cc-nav-brand">Intelligent Capital Call & Liquidity Orchestration</div>
        <div style={{ display: 'flex', gap: '1rem', alignItems: 'center' }}>
          <div className="cc-search-container">
            <svg className="cc-search-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
            <input 
              type="text" 
              placeholder="Search capital calls..." 
              className="cc-search-input"
              value={searchQuery}
              onChange={e => {
                setSearchQuery(e.target.value);
                setCurrentPage(1);
              }}
            />
          </div>
          <button 
            onClick={() => navigate('/')}
            style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#6b7280', fontSize: '0.875rem', fontWeight: 500 }}
          >
            Log Out
          </button>
        </div>
      </nav>

      <main className="cc-main-content">
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

        <div className="cc-stats-grid">
          {statCards.map((stat, idx) => (
            <div key={idx} className="cc-stat-card">
              <div className="cc-stat-label">{stat.label}</div>
              <div className="cc-stat-value" style={stat.valueColor ? { color: stat.valueColor } : {}}>{stat.value}</div>
            </div>
          ))}
        </div>

        <div className="cc-table-container">
          <div className="cc-table-tabs">
            {['All Calls', 'Active', 'Pending', 'Settled'].map(tab => (
              <button 
                key={tab} 
                className={`cc-tab ${activeTab === tab ? 'active' : ''}`}
                onClick={() => {
                  setActiveTab(tab);
                  setCurrentPage(1);
                }}
              >
                {tab}
              </button>
            ))}
          </div>

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
                {paginatedCalls.length > 0 ? paginatedCalls.map((call) => (
                  <CapitalCallRow key={call.id} call={call} />
                )) : (
                  <tr>
                    <td colSpan={7} style={{ textAlign: 'center', padding: '2rem', color: '#6b7280' }}>
                      No capital calls found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {searchedCalls.length > 0 && (
            <div className="cc-pagination">
              <div className="cc-pagination-info">
                Showing {startIndex + 1}-{Math.min(startIndex + itemsPerPage, searchedCalls.length)} of {searchedCalls.length} Capital Calls
              </div>
              <div className="cc-pagination-controls">
                <button 
                  className="cc-page-btn cc-page-arrow" 
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={validPage === 1}
                  style={{ opacity: validPage === 1 ? 0.5 : 1, cursor: validPage === 1 ? 'not-allowed' : 'pointer' }}
                >
                  &lt;
                </button>
                
                {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                  <button 
                    key={page} 
                    className={`cc-page-btn ${validPage === page ? 'active' : ''}`}
                    onClick={() => setCurrentPage(page)}
                  >
                    {page}
                  </button>
                ))}

                <button 
                  className="cc-page-btn cc-page-arrow"
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={validPage === totalPages}
                  style={{ opacity: validPage === totalPages ? 0.5 : 1, cursor: validPage === totalPages ? 'not-allowed' : 'pointer' }}
                >
                  &gt;
                </button>
              </div>
            </div>
          )}
        </div>
      </main>

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
