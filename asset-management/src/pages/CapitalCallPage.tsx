import { useState } from 'react';
import '../styles/CapitalCallPage.css';

const CapitalCallPage = () => {
  const [activeTab, setActiveTab] = useState('All Calls');

  //!!!placeholder vars (static)
  const stats = [
    { label: 'TOTAL CALLED (YTD)', value: '$42.8M' },
    { label: 'PENDING LIQUIDITY', value: '$12.4M', valueColor: '#4f46e5' },
    { label: 'AVG. LP RESPONSE', value: '4.2 Days' },
    { label: 'ACTIVE CALLS', value: '07' },
  ];

  //!!!temporary list of fake CCs
  const calls = [
    {
      id: 'CC-2024-008',
      fund: 'Growth Fund IV',
      target: '$5,000,000.00',
      received: '$3,500,000.00',
      progress: 70,
      progressColor: '#4f46e5',
      lpCompletion: '7 / 10',
      deadlineDate: 'Oct 24, 2024',
      deadlineSub: '4 Days Remaining',
      deadlineSubColor: '#dc2626',
      status: 'ACTIVE',
      statusType: 'active'
    },
    {
      id: 'CC-2024-009',
      fund: 'Tech Ventures II',
      target: '$2,250,000.00',
      received: '$0.00',
      progress: 0,
      progressColor: '#e5e7eb',
      lpCompletion: '0 / 12',
      deadlineDate: 'Nov 05, 2024',
      deadlineSub: 'Scheduled',
      deadlineSubColor: '#6b7280',
      status: 'WAITING',
      statusType: 'waiting'
    },
    {
      id: 'CC-2024-007',
      fund: 'Real Estate Alpha',
      target: '$12,000,000.00',
      received: '$8,400,000.00',
      progress: 70,
      progressColor: '#d97706',
      lpCompletion: '15 / 18',
      deadlineDate: 'Oct 18, 2024',
      deadlineSub: 'Bridge Financing',
      deadlineSubColor: '#6b7280',
      status: 'BRIDGE ACTIVE',
      statusType: 'bridge'
    },
    {
      id: 'CC-2024-006',
      fund: 'Opportunities Fund I',
      target: '$8,500,000.00',
      received: '$8,500,000.00',
      progress: 100,
      progressColor: '#16a34a',
      lpCompletion: '24 / 24',
      deadlineDate: 'Oct 01, 2024',
      deadlineSub: 'Closed',
      deadlineSubColor: '#16a34a',
      status: 'COMPLETED',
      statusType: 'completed'
    }
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
          <button className="cc-new-call-btn">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="16"></line><line x1="8" y1="12" x2="16" y2="12"></line></svg>
            Start New Capital Call
          </button>
        </header>

        {/* Stats Section */}
        <div className="cc-stats-grid">
          {stats.map((stat, idx) => (
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
                {calls.map((call, idx) => (
                  <tr key={idx}>
                    <td>
                      <div className="cc-cell-title">{call.id}</div>
                      <div className="cc-cell-sub">{call.fund}</div>
                    </td>
                    <td className="cc-target-amount">{call.target}</td>
                    <td>
                      <div className="cc-received-amount" style={{color: call.progressColor === '#e5e7eb' ? '#111827' : call.progressColor}}>{call.received}</div>
                      <div className="cc-progress-bar-bg">
                        <div className="cc-progress-bar-fill" style={{width: `${call.progress}%`, backgroundColor: call.progressColor}}></div>
                      </div>
                    </td>
                    <td className="cc-lp-completion" style={{color: call.progressColor === '#e5e7eb' ? '#6b7280' : call.progressColor}}>{call.lpCompletion}</td>
                    <td>
                      <div className="cc-cell-title">{call.deadlineDate}</div>
                      <div className="cc-cell-sub" style={{color: call.deadlineSubColor}}>{call.deadlineSub}</div>
                    </td>
                    <td>
                      <span className={`cc-badge cc-badge-${call.statusType}`}>
                        {call.status}
                      </span>
                    </td>
                    <td className="cc-action-cell">
                      <button className="cc-action-btn">
                        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="1"></circle><circle cx="12" cy="5" r="1"></circle><circle cx="12" cy="19" r="1"></circle></svg>
                      </button>
                    </td>
                  </tr>
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
    </div>
  );
};

export default CapitalCallPage;
