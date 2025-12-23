import { useState, useRef, useEffect } from 'react';
import { uploadStatement, getStatements, Statement, ImportResult } from '../api/client';

export default function Import() {
  const [statements, setStatements] = useState<Statement[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadStatements = async () => {
    setLoading(true);
    try {
      const data = await getStatements();
      setStatements(data);
    } catch (e) {
      console.error(e);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadStatements();
  }, []);

  const handleUpload = async (file: File) => {
    const lowerName = file.name.toLowerCase();
    if (!lowerName.endsWith('.pdf') && !lowerName.endsWith('.csv')) {
      setResult({
        success: false,
        statement_id: null,
        transactions_imported: 0,
        transactions_skipped: 0,
        errors: ['Only PDF or CSV files are supported'],
        message: 'Please upload a PDF or CSV file',
      });
      return;
    }

    setUploading(true);
    setResult(null);

    try {
      const res = await uploadStatement(file);
      setResult(res);
      if (res.success) {
        loadStatements();
      }
    } catch (e) {
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

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Statement Import</p>
          <h1>Bring in your PDFs and CSVs.</h1>
        </div>
        <div className="page-note">CSV files require "starling" in the filename.</div>
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
            <p>Uploading and parsing...</p>
          </div>
        ) : (
          <>
            <svg className="upload-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M12 12l-3 3m0 0l3-3m-3 3h8" />
            </svg>
            <p className="upload-title">Drop your statement PDF or CSV here</p>
            <p className="upload-sub">HSBC PDFs, Starling CSVs</p>
            <button
              className="btn-primary"
              onClick={() => fileInputRef.current?.click()}
            >
              Select file
            </button>
          </>
        )}
      </div>

      {/* Result */}
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

      {/* Previous Imports */}
      <div className="panel table-card">
        <div className="panel-header">
          <div>
            <div className="panel-title">Import history</div>
            <div className="panel-sub">Latest statement files</div>
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
                <tr key={stmt.id}>
                  <td className="table-cell">{stmt.filename}</td>
                  <td className="table-cell">
                    {formatDate(stmt.period_start)} - {formatDate(stmt.period_end)}
                  </td>
                  <td className="table-cell right">{stmt.transaction_count}</td>
                  <td className="table-cell right">
                    {stmt.closing_balance !== null ? `£${stmt.closing_balance.toFixed(2)}` : '-'}
                  </td>
                  <td className="table-cell">{formatDate(stmt.imported_at)}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
