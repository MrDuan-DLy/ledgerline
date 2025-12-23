import { useEffect, useRef, useState } from 'react';
import {
  uploadReceipts,
  getReceipts,
  confirmReceipt,
  getCategories,
  Receipt,
  ReceiptUploadResult,
  Category,
} from '../api/client';

const formatDate = (dateStr?: string | null) => {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
};

export default function Receipts() {
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [overrides, setOverrides] = useState<Record<number, {
    merchant_name?: string;
    receipt_date?: string;
    total_amount?: number;
    category_id?: number;
  }>>({});
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [results, setResults] = useState<ReceiptUploadResult[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadReceipts = async () => {
    setLoading(true);
    try {
      const [receiptData, categoryData] = await Promise.all([
        getReceipts(),
        getCategories(),
      ]);
      setReceipts(receiptData);
      setCategories(categoryData);
    } catch (e) {
      console.error(e);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadReceipts();
  }, []);

  const handleUpload = async (files: File[]) => {
    const invalid = files.filter((file) => {
      const lower = file.name.toLowerCase();
      return !lower.endsWith('.jpg') && !lower.endsWith('.jpeg') && !lower.endsWith('.png');
    });
    if (invalid.length) {
      setResults([
        {
          success: false,
          receipt_id: null,
          message: 'Please upload JPG or PNG images',
          errors: ['Only .jpg, .jpeg, .png are supported'],
        },
      ]);
      return;
    }

    setUploading(true);
    setResults([]);

    try {
      const res = await uploadReceipts(files);
      setResults(res);
      if (res.some((item) => item.success)) {
        loadReceipts();
      }
    } catch (e) {
      setResults([
        {
          success: false,
          receipt_id: null,
          message: 'Failed to upload receipts',
          errors: ['Upload failed'],
        },
      ]);
    }

    setUploading(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    if (e.dataTransfer.files?.length) {
      handleUpload(Array.from(e.dataTransfer.files));
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files?.length) {
      handleUpload(Array.from(e.target.files));
    }
  };

  const handleConfirm = async (receipt: Receipt) => {
    const payload = overrides[receipt.id] || {
      merchant_name: receipt.merchant_name || undefined,
      receipt_date: receipt.receipt_date || undefined,
      total_amount: receipt.total_amount ?? undefined,
    };
    await confirmReceipt(receipt.id, payload);
    loadReceipts();
  };

  const handleLinkMatch = async (receipt: Receipt) => {
    if (!receipt.matched_transaction_id) return;
    await confirmReceipt(receipt.id, { transaction_id: receipt.matched_transaction_id });
    loadReceipts();
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Receipt OCR</p>
          <h1>Turn receipts into transactions.</h1>
        </div>
        <div className="page-note">Uploads are parsed with Gemini, then you confirm.</div>
      </div>

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
          accept=".jpg,.jpeg,.png"
          multiple
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
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 16v-6m0 0l-3 3m3-3l3 3M6 20h12" />
            </svg>
            <p className="upload-title">Drop receipt image here</p>
            <p className="upload-sub">JPG or PNG, upright and clear</p>
            <button
              className="btn-primary"
              onClick={() => fileInputRef.current?.click()}
            >
              Select image
            </button>
          </>
        )}
      </div>

      {results.length > 0 && (
        <div className="panel result-card">
          <div className="result-title">Batch results</div>
          <ul className="result-errors">
            {results.map((res, i) => (
              <li key={i} className={res.success ? 'result-ok' : 'result-bad'}>
                {res.success ? 'Parsed' : 'Failed'}: {res.message}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="panel">
        <div className="panel-header">
          <div>
            <div className="panel-title">Receipts</div>
            <div className="panel-sub">Confirm to create a transaction</div>
          </div>
          <div className="panel-pill">{loading ? 'Loading' : `${receipts.length} files`}</div>
        </div>

        <div className="receipt-list">
          {loading ? (
            <div className="empty">Loading receipts...</div>
          ) : receipts.length === 0 ? (
            <div className="empty">No receipts uploaded yet.</div>
          ) : (
            receipts.map((receipt) => (
              <div key={receipt.id} className="receipt-card">
                <div>
                  <div className="receipt-title">{receipt.merchant_name || 'Unknown merchant'}</div>
                  <div className="receipt-meta">
                    {formatDate(receipt.receipt_date)} · {receipt.total_amount ? `£${receipt.total_amount.toFixed(2)}` : '-'}
                  </div>
                </div>
                <div className="receipt-status">{receipt.status}</div>
                {receipt.matched_transaction_id && receipt.status !== 'confirmed' && (
                  <div className="receipt-match">
                    Suggested match: {receipt.matched_transaction_description || 'Transaction'} ·{' '}
                    {receipt.matched_transaction_date ? formatDate(receipt.matched_transaction_date) : '-'} ·{' '}
                    {receipt.matched_transaction_amount !== null ? `£${receipt.matched_transaction_amount?.toFixed(2)}` : '-'}
                    {receipt.matched_reason ? ` (${receipt.matched_reason})` : ''}
                    <button
                      className="btn-secondary"
                      onClick={() => handleLinkMatch(receipt)}
                    >
                      Link match
                    </button>
                  </div>
                )}
                <div className="receipt-actions">
                  {receipt.status !== 'confirmed' && (
                    <>
                      <input
                        type="text"
                        className="table-select"
                        placeholder="Merchant"
                        defaultValue={receipt.merchant_name || ''}
                        onBlur={(e) => {
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: {
                              ...prev[receipt.id],
                              merchant_name: e.target.value || undefined,
                            },
                          }));
                        }}
                      />
                      <input
                        type="date"
                        className="table-select"
                        defaultValue={receipt.receipt_date || ''}
                        onBlur={(e) => {
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: {
                              ...prev[receipt.id],
                              receipt_date: e.target.value || undefined,
                            },
                          }));
                        }}
                      />
                      <input
                        type="number"
                        step="0.01"
                        className="table-select"
                        placeholder="Total"
                        defaultValue={receipt.total_amount ?? ''}
                        onBlur={(e) => {
                          const value = e.target.value ? Number(e.target.value) : undefined;
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: {
                              ...prev[receipt.id],
                              total_amount: value,
                            },
                          }));
                        }}
                      />
                      <select
                        className="table-select"
                        defaultValue=""
                        onChange={(e) => {
                          const value = e.target.value ? Number(e.target.value) : undefined;
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: {
                              ...prev[receipt.id],
                              category_id: value,
                            },
                          }));
                        }}
                      >
                        <option value="">Category</option>
                        {categories.filter(c => c.is_expense).map((cat) => (
                          <option key={cat.id} value={cat.id}>{cat.name}</option>
                        ))}
                      </select>
                      <button
                        className="btn-primary"
                        onClick={() => handleConfirm(receipt)}
                      >
                        Confirm
                      </button>
                    </>
                  )}
                  {receipt.status === 'confirmed' && (
                    <div className="receipt-confirmed">Linked to transaction #{receipt.transaction_id}</div>
                  )}
                </div>
                {receipt.items.length > 0 && (
                  <div className="receipt-items">
                    {receipt.items.map((item) => (
                      <div key={item.id} className="receipt-item">
                        <span>{item.name}</span>
                        <span>{item.line_total ? `£${item.line_total.toFixed(2)}` : ''}</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
