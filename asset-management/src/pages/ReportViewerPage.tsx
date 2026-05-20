import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import html2pdf from 'html2pdf.js';
import '../styles/ReportViewerPage.css';

interface LPDetail {
  lpId: string;
  status: string;
  amountUSD?: number;
  riskScore?: number;
}

interface ReportData {
  reportType: string;
  generatedAt: string;
  callId: string;
  fundId: string;
  summary: {
    targetAmountUSD: number;
    totalCommitted: number;
    gapUSD: number;
    gapPercent: number;
    bridgeTriggered: boolean;
  };
  lpDetails: LPDetail[];
  portfolioRisk: any;
  bridgeResult: any;
}

const currencyFmt = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 0,
});

const compactCurrencyFmt = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  notation: 'compact',
  maximumFractionDigits: 2,
});

export default function ReportViewerPage() {
  const { callId } = useParams();
  const navigate = useNavigate();
  const [report, setReport] = useState<ReportData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    fetch(`/reports/${callId}.json`)
      .then(res => {
        if (!res.ok) throw new Error('Report not found or not yet generated');
        return res.json();
      })
      .then(data => {
        setReport(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.message);
        setLoading(false);
      });
  }, [callId]);

  if (loading) return <div className="report-loading">Loading Institutional Report...</div>;
  if (error) return (
    <div className="report-error">
      <div className="report-error-content">
        <h2>Report Not Available</h2>
        <p>{error}</p>
        <button onClick={() => navigate('/capital-calls')} className="report-back-btn">Return to Dashboard</button>
      </div>
    </div>
  );

  if (!report) return null;

  const dateObj = new Date(report.generatedAt);
  const formattedDate = dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' }) + ' - ' + dateObj.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' });

  const isGap = report.summary.gapUSD > 0;
  const committedPercent = Math.max(0, 100 - report.summary.gapPercent);

  const handleExportPDF = () => {
    const element = document.getElementById('report-document');
    if (!element) return;
    const opt = {
      margin: 0.5,
      filename: `Capital_Call_Report_${callId}.pdf`,
      image: { type: 'jpeg' as const, quality: 0.98 },
      html2canvas: { scale: 2 },
      jsPDF: { unit: 'in' as const, format: 'letter' as const, orientation: 'portrait' as const }
    };
    html2pdf().set(opt).from(element).save();
  };

  return (
    <div className="report-page-container">
      {/* Top action bar - Hidden during print */}
      <div className="report-action-bar no-print">
        <button className="report-back-btn" onClick={() => navigate('/capital-calls')}>
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
          Back to Dashboard
        </button>
        <div className="report-actions">
          <button className="report-btn report-btn-primary" onClick={handleExportPDF}>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
            Export PDF
          </button>
        </div>
      </div>

      {/* Printable Report Document */}
      <div id="report-document" className="report-document">
        
        {/* Header */}
        <div className="report-header">
          <div className="report-title-section">
            <h1>LIQUIDITY REPORT</h1>
            <p>Capital Call Orchestration Summary</p>
          </div>
          <div className="report-confidential-badge">
            <div className="bank-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="10" width="18" height="12" rx="2"/><path d="M3 6h18"/><path d="M12 2v4"/><path d="M8 2v4"/><path d="M16 2v4"/></svg>
            </div>
            <span>CONFIDENTIAL DATA</span>
          </div>
        </div>

        {/* Info Grid */}
        <div className="report-info-grid">
          <div className="report-info-col">
            <label>CALL ID</label>
            <strong>{report.callId}</strong>
          </div>
          <div className="report-info-col">
            <label>FUND ID</label>
            <strong>{report.fundId}</strong>
          </div>
          <div className="report-info-col">
            <label>GENERATED</label>
            <strong>{formattedDate}</strong>
          </div>
        </div>

        {/* Summary Metrics */}
        <div className="report-metrics-grid">
          <div className="report-metric-card">
            <label>Target Amount</label>
            <div className="metric-value">{compactCurrencyFmt.format(report.summary.targetAmountUSD)}</div>
          </div>
          <div className="report-metric-card">
            <label>Total Committed</label>
            <div className="metric-value">{compactCurrencyFmt.format(report.summary.totalCommitted)}</div>
          </div>
          <div className={`report-metric-card ${isGap ? 'metric-alert' : ''}`}>
            <label>Gap Amount</label>
            <div className="metric-value">{compactCurrencyFmt.format(report.summary.gapUSD)}</div>
          </div>
          <div className="report-metric-card">
            <label>Gap %</label>
            <div className="metric-value">{report.summary.gapPercent.toFixed(1)}%</div>
          </div>
          <div className="report-metric-card">
            <label>Bridge Active</label>
            <div className="metric-badge-container">
              <span className={`report-badge ${report.summary.bridgeTriggered ? 'badge-yes' : 'badge-no'}`}>
                {report.summary.bridgeTriggered ? 'YES' : 'NO'}
              </span>
            </div>
          </div>
        </div>

        {/* Portfolio Risk */}
        {report.portfolioRisk && (
          <div className="report-velocity-section" style={{ borderTop: 'none', paddingTop: '0', marginTop: '-0.5rem' }}>
            <div className="velocity-info" style={{ width: '100%' }}>
              <h3>Portfolio Risk Summary</h3>
              <p>
                Concentration Score (HHI): <strong>{report.portfolioRisk.concentrationScore}</strong> | Top Risky LPs: <strong>{report.portfolioRisk.topRiskyLPs?.join(', ') || 'None'}</strong>
              </p>
            </div>
          </div>
        )}

        {/* Funding Velocity */}
        <div className="report-velocity-section">
          <div className="velocity-info">
            <h3>Funding Velocity</h3>
            <p>
              The current call cycle is operating at {committedPercent.toFixed(1)}% commitment efficiency.
              {report.summary.bridgeTriggered ? ' Gap coverage is currently mitigated via the alpha-line credit facility.' : ' Target successfully met.'}
            </p>
          </div>
          <div className="velocity-bar-container">
            <div className="velocity-bar-track">
              <div className="velocity-bar-fill" style={{ width: `${committedPercent}%` }}></div>
            </div>
            <div className="velocity-bar-labels">
              <span>Committed</span>
              <span>{report.summary.gapPercent.toFixed(1)}% Shortfall</span>
            </div>
          </div>
        </div>

        {/* LP Responses Table */}
        <div className="report-table-section">
          <div className="table-header-flex">
            <h3>Institutional LP Responses</h3>
            <div className="table-legend">
              <span className="legend-item"><span className="dot dot-committed"></span> Committed</span>
              <span className="legend-item"><span className="dot dot-pending"></span> Pending</span>
              <span className="legend-item"><span className="dot dot-defaulted"></span> Defaulted</span>
            </div>
          </div>
          <table className="report-lp-table">
            <thead>
              <tr>
                <th>LP ID</th>
                <th>STATUS</th>
                <th className="align-right">CONTRIBUTION</th>
                <th className="align-right">RISK SCORE</th>
              </tr>
            </thead>
            <tbody>
              {report.lpDetails.map((lp, idx) => (
                <tr key={idx} className={`row-${lp.status.toLowerCase()}`}>
                  <td><strong>{lp.lpId}</strong></td>
                  <td>
                    <span className={`status-text text-${lp.status.toLowerCase()}`}>
                      {lp.status.charAt(0).toUpperCase() + lp.status.slice(1)}
                    </span>
                  </td>
                  <td className="align-right amount-col">
                    {lp.status === 'defaulted' ? <span className="amount-alert">{currencyFmt.format(lp.amountUSD || 0)}</span> : currencyFmt.format(lp.amountUSD || 0)}
                  </td>
                  <td className="align-right risk-col">
                    <span className={`risk-val ${((lp.riskScore||0)*10) > 7 ? 'risk-high' : ((lp.riskScore||0)*10) > 4 ? 'risk-med' : 'risk-low'}`}>
                      {((lp.riskScore || 0) * 10).toFixed(1)}/10
                    </span>
                    <span className="risk-indicator"></span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Footer */}
        <div className="report-footer">
          <div className="signature-section">
            <div className="signature-line">
              <span>/ Fund Manager Digital Signature /</span>
              <strong>{dateObj.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</strong>
            </div>
          </div>
          
          <div className="footer-meta-grid">
            <div className="footer-meta-block">
              <strong>AUTHORIZED SIGNATURE</strong>
              <span>Compliance Officer - Quest1</span>
            </div>
            <div className="footer-meta-block">
              <strong>EFFECTIVE DATE</strong>
              <span>Regulatory Validation Stamp: #{(Math.random()*90000+10000).toFixed(0)}-A</span>
            </div>
          </div>

          <div className="footer-bottom-row">
            <span>Institutional Liquidity Services | Proprietary & Confidential</span>
            <span>Ref ID: {report.fundId}-{report.callId} • <strong>Page 1 of 1</strong></span>
          </div>
          <div className="footer-disclaimer">
            This document is an official record. Unauthorized distribution is prohibited under the SEC Disclosure Act.
          </div>
        </div>

      </div>
    </div>
  );
}
