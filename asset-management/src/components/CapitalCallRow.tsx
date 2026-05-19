import type { FC } from 'react';
import { useNavigate } from 'react-router-dom';

export interface CapitalCall {
  id: string;
  fund: string;
  target: number;
  received: number;
  lpCompletion: string;
  deadlineDate: string;
  status: string;
}

interface CapitalCallRowProps {
  call: CapitalCall;
}

const CapitalCallRow: FC<CapitalCallRowProps> = ({ call }) => {
  const navigate = useNavigate();
  const targetFmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(call.target);
  const receivedFmt = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(call.received);
  
  const progress = call.target > 0 ? (call.received / call.target) * 100 : 0;
  
  let statusType = 'waiting';
  let progressColor = '#4f46e5';
  let displayStatus = call.status.toUpperCase();

  if (call.status === 'issued') { 
    statusType = 'active'; 
    progressColor = '#4f46e5'; 
    displayStatus = 'ACTIVE';
  } else if (call.status === 'settled') { 
    statusType = 'completed'; 
    progressColor = '#16a34a'; 
    displayStatus = 'COMPLETED';
  } else {
    statusType = 'waiting';
    progressColor = '#e5e7eb';
  }

  // calculate deadline
  const deadline = new Date(call.deadlineDate);
  const diffDays = Math.ceil((deadline.getTime() - new Date().getTime()) / (1000 * 3600 * 24));
  let deadlineSub = diffDays > 0 ? `${diffDays} Days Remaining` : 'Past Due';
  let deadlineSubColor = diffDays > 0 ? '#6b7280' : '#dc2626';

  const deadlineDateStr = deadline.toLocaleDateString('en-US', { month: 'short', day: '2-digit', year: 'numeric' });

  const handleRowClick = () => {
    navigate(`/capital-calls/${call.id}`);
  };

  return (
    <tr
      onClick={handleRowClick}
      style={{ cursor: 'pointer' }}
      className="cc-table-row-clickable"
    >
      <td>
        <div className="cc-cell-title">{call.id}</div>
        <div className="cc-cell-sub">{call.fund}</div>
      </td>
      <td className="cc-target-amount">{targetFmt}</td>
      <td>
        <div className="cc-received-amount" style={{color: progressColor === '#e5e7eb' ? '#111827' : progressColor}}>
          {receivedFmt}
        </div>
        <div className="cc-progress-bar-bg">
          <div className="cc-progress-bar-fill" style={{width: `${progress}%`, backgroundColor: progressColor}}></div>
        </div>
      </td>
      <td className="cc-lp-completion" style={{color: progressColor === '#e5e7eb' ? '#6b7280' : progressColor}}>
        {call.lpCompletion}
      </td>
      <td>
        <div className="cc-cell-title">{deadlineDateStr}</div>
        <div className="cc-cell-sub" style={{color: deadlineSubColor}}>{deadlineSub}</div>
      </td>
      <td>
        <span className={`cc-badge cc-badge-${statusType}`}>
          {displayStatus}
        </span>
      </td>
      <td className="cc-action-cell" onClick={e => e.stopPropagation()}>
        <button
          className="cc-action-btn"
          style={{ width: 'auto', padding: '0.25rem 0.75rem', fontSize: '0.75rem', background: '#eef2ff', borderRadius: '4px', fontWeight: 600, color: '#4f46e5', border: '1px solid #c7d2fe' }}
          onClick={() => navigate(`/capital-calls/${call.id}`)}
        >
          View Details
        </button>
      </td>
    </tr>
  );
};

export default CapitalCallRow;
