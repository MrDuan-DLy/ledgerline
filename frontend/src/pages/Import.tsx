import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  uploadStatement,
  uploadForReview,
  getStatements,
  getImportSessions,
  deleteImportSession,
  Statement,
  ImportResult,
  ImportUploadResult,
  ImportSession,
} from '../api/client';
import { formatCurrency } from '../utils/format';
import { useAppConfig } from '../contexts/AppConfig';

export default function Import() {
  const navigate = useNavigate();
  const [statements, setStatements] = useState<Statement[]>([]);
  const [sessions, setSessions] = useState<ImportSession[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const [stmts, sess] = await Promise.all([
        getStatements(),
        getImportSessions(),
      ]);
      setStatements(stmts);
      setSessions(sess);
    } catch (e) {
      console.error(e);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleUpload = async (file: File) => {
    const lowerName = file.name.toLowerCase();
    const isPdf = lowerName.endsWith('.pdf');
    const isCsv = lowerName.endsWith('.csv');

    if (!isPdf && !isCsv) {
      setError('Upload a bank statement (PDF or CSV). For receipt images, use the Receipts page.');
      return;
    }

    setUploading(true);
    setResult(null);
    setError(null);

    // PDFs go through AI review flow
    if (isPdf) {
      try {
        const res: ImportUploadResult = await uploadForReview(file);
        if (res.success && res.session_id) {
          navigate(`/review/${res.session_id}`);
          return;
        }
        setError(res.message || 'Upload failed');
        if (res.errors?.length) {
          setError(res.errors.join(', '));
        }
      } catch {
        setError('Failed to upload file');
      }
      setUploading(false);
      return;
    }

    // CSVs go through the existing direct import flow
    try {
      const res = await uploadStatement(file);
      setResult(res);
      if (res.success) {
        loadData();
      }
    } catch {
      setResult({
        success: false,
        statement_id: null,
        transactions_imported: 0,
        transactions_skipped: 0,
        errors: ['Upload failed'],
        message: 'Failed to upload file',
      });
    }

    setUploading(false);
  };

  const handleDeleteSession = async (sessionId: string) => {
    await deleteImportSession(sessionId);
    setSessions(prev => prev.filter(s => s.id !== sessionId));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    if (e.dataTransfer.files?.[0]) {
      handleUpload(e.dataTransfer.files[0]);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files?.[0]) {
      handleUpload(e.target.files[0]);
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    });
  };

  const statementSessions = sessions.filter(s => s.source_type !== 'receipt_image');
  const pendingSessions = statementSessions.filter(s => s.status === 'pending');
  const confirmedSessions = statementSessions.filter(s => s.status === 'confirmed');

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Statement Import</p>
          <h1>Import bank statements.</h1>
        </div>
        <div className="page-note">PDFs use AI extraction with review. CSVs import directly.</div>
      </div>

      {/* Upload Area */}
      <div
        className={`panel upload-drop ${dragActive ? 'active' : ''}`}
        onDragOver={(e) => { e.preventDefault(); setDragActive(true); }}
        onDragLeave={() => setDragActive(false)}
        onDrop={handleDrop}
      >
        <input
          type="file"
          ref={fileInputRef}
          className="hidden"
          accept=".pdf,.csv"
          onChange={handleFileSelect}
        />

        {uploading ? (
          <div className="upload-state">
            <div className="spinner" />
            <p>Uploading and extracting with AI...</p>
          </div>
        ) : (
          <>
            <svg className="upload-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M12 12l-3 3m0 0l3-3m-3 3h8" />
            </svg>
            <p className="upload-title">Drop your bank statement here</p>
            <p className="upload-sub">Bank statement PDFs or CSVs</p>
            <button
              className="btn-primary"
              onClick={() => fileInputRef.current?.click()}
            >
              Select file
            </button>
          </>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="panel result-card error">
          <div className="result-title">Upload Failed</div>
          <p className="result-message">{error}</p>
        </div>
      )}

      {/* Result (CSV direct import) */}
      {result && (
        <div className={`panel result-card ${result.success ? 'success' : 'error'}`}>
          <div className="result-title">
            {result.success ? 'Import Successful' : 'Import Failed'}
          </div>
          <p className="result-message">{result.message}</p>
          {result.success && (
            <div className="result-meta">
              {result.transactions_imported} transactions imported, {result.transactions_skipped} duplicates skipped
            </div>
          )}
          {result.errors.length > 0 && (
            <ul className="result-errors">
              {result.errors.map((e, i) => <li key={i}>{e}</li>)}
            </ul>
          )}
        </div>
      )}

      {/* Pending Review Sessions */}
      {pendingSessions.length > 0 && (
        <div className="panel table-card">
          <div className="panel-header" style={{ padding: '16px 16px 0' }}>
            <div>
              <div className="panel-title">Pending Review</div>
              <div className="panel-sub">Uploads waiting for your review</div>
            </div>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th className="table-head">File</th>
                <th className="table-head">Type</th>
                <th className="table-head right">Items</th>
                <th className="table-head">Uploaded</th>
                <th className="table-head right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pendingSessions.map((sess) => (
                <tr key={sess.id}>
                  <td className="table-cell">{sess.source_file}</td>
                  <td className="table-cell">
                    <span className="panel-pill">
                      {sess.source_type === 'pdf' ? 'PDF' : 'Receipt'}
                    </span>
                  </td>
                  <td className="table-cell right">{sess.items.length}</td>
                  <td className="table-cell">{formatDate(sess.created_at)}</td>
                  <td className="table-cell right">
                    <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                      <button
                        className="btn-primary"
                        style={{ padding: '4px 12px', fontSize: 13 }}
                        onClick={() => navigate(`/review/${sess.id}`)}
                      >
                        Review
                      </button>
                      <button
                        className="btn-secondary"
                        style={{ padding: '4px 12px', fontSize: 13, color: 'var(--accent-rose)' }}
                        onClick={() => handleDeleteSession(sess.id)}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Previous Imports (confirmed statements) */}
      <div className="panel table-card">
        <div className="panel-header" style={{ padding: '16px 16px 0' }}>
          <div>
            <div className="panel-title">Import history</div>
            <div className="panel-sub">Confirmed statement files</div>
          </div>
        </div>
        <table className="table">
          <thead>
            <tr>
              <th className="table-head">Filename</th>
              <th className="table-head">Period</th>
              <th className="table-head right">Transactions</th>
              <th className="table-head right">Balance</th>
              <th className="table-head">Imported</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="table-empty">Loading...</td></tr>
            ) : statements.length === 0 ? (
              <tr><td colSpan={5} className="table-empty">No statements imported yet</td></tr>
            ) : (
              statements.map((stmt) => (
                <tr key={stmt.id} className="table-row-link" onClick={() => navigate(`/transactions?statement_id=${stmt.id}`)}>
                  <td className="table-cell">{stmt.filename}</td>
                  <td className="table-cell">
                    {formatDate(stmt.period_start)} - {formatDate(stmt.period_end)}
                  </td>
                  <td className="table-cell right">{stmt.transaction_count}</td>
                  <td className="table-cell right">
                    {stmt.closing_balance !== null ? formatCurrency(stmt.closing_balance) : '-'}
                  </td>
                  <td className="table-cell">{formatDate(stmt.imported_at)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Confirmed AI Import Sessions */}
      {confirmedSessions.length > 0 && (
        <div className="panel table-card">
          <div className="panel-header" style={{ padding: '16px 16px 0' }}>
            <div>
              <div className="panel-title">AI import history</div>
              <div className="panel-sub">Previously reviewed imports</div>
            </div>
          </div>
          <table className="table">
            <thead>
              <tr>
                <th className="table-head">File</th>
                <th className="table-head">Type</th>
                <th className="table-head right">Items</th>
                <th className="table-head">Imported</th>
                <th className="table-head right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {confirmedSessions.map((sess) => (
                <tr key={sess.id} className="table-row-link" onClick={() => navigate(`/review/${sess.id}`)}>
                  <td className="table-cell">{sess.source_file}</td>
                  <td className="table-cell">
                    <span className="panel-pill">
                      {sess.source_type === 'pdf' ? 'PDF' : 'CSV'}
                    </span>
                  </td>
                  <td className="table-cell right">{sess.items.length}</td>
                  <td className="table-cell">{formatDate(sess.created_at)}</td>
                  <td className="table-cell right">
                    <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                      <button
                        className="btn-secondary"
                        style={{ padding: '4px 12px', fontSize: 13 }}
                        onClick={(e) => { e.stopPropagation(); navigate(`/review/${sess.id}`); }}
                      >
                        View
                      </button>
                      <button
                        className="btn-secondary"
                        style={{ padding: '4px 12px', fontSize: 13, color: 'var(--accent-rose)' }}
                        onClick={(e) => { e.stopPropagation(); handleDeleteSession(sess.id); }}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
