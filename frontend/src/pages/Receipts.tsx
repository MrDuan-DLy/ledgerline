import { useEffect, useRef, useState } from 'react';
import {
  uploadReceipts,
  getReceipts,
  confirmReceipt,
  getCategories,
  receiptImageUrl,
  Receipt,
  ReceiptUploadResult,
  Category,
} from '../api/client';
import { useAppConfig } from '../contexts/AppConfig';

const formatDate = (dateStr?: string | null, locale = 'en-GB') => {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleDateString(locale, {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
};

const PAGE_SIZE = 10;

export default function Receipts() {
  const { currencySymbol } = useAppConfig();
  const [allReceipts, setAllReceipts] = useState<Receipt[]>([]);
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
  const [page, setPage] = useState(1);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const totalPages = Math.max(1, Math.ceil(allReceipts.length / PAGE_SIZE));
  const receipts = allReceipts.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const loadReceipts = async () => {
    setLoading(true);
    try {
      const [receiptData, categoryData] = await Promise.all([
        getReceipts(),
        getCategories(),
      ]);
      setAllReceipts(receiptData);
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
      return !lower.endsWith('.jpg') && !lower.endsWith('.jpeg') && !lower.endsWith('.png') && !lower.endsWith('.webp');
    });
    if (invalid.length) {
      setResults([
        {
          success: false,
          receipt_id: null,
          message: 'Please upload JPG, PNG, or WebP images',
          errors: ['Only .jpg, .jpeg, .png, .webp are supported'],
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
    } catch {
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
          accept=".jpg,.jpeg,.png,.webp"
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
            <p className="upload-sub">JPG, PNG, or WebP — upright and clear</p>
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
            <div className="panel-sub">Compare original image with OCR results, then confirm</div>
          </div>
          <div className="panel-pill">{loading ? 'Loading' : `${allReceipts.length} files`}</div>
        </div>

        <div className="receipt-list">
          {loading ? (
            <div className="empty">Loading receipts...</div>
          ) : receipts.length === 0 ? (
            <div className="empty">No receipts uploaded yet.</div>
          ) : (
            receipts.map((receipt) => (
              <div key={receipt.id} className="receipt-compare-card">
                {/* Left: Original Image */}
                <div className="receipt-compare-image">
                  <img
                    src={receiptImageUrl(receipt.id)}
                    alt={receipt.merchant_name || 'Receipt'}
                  />
                </div>

                {/* Right: OCR Results */}
                <div className="receipt-compare-details">
                  <div className="receipt-compare-header">
                    <h3 className="receipt-compare-merchant">
                      {receipt.merchant_name || 'Unknown merchant'}
                    </h3>
                    <span className={`receipt-compare-status status-${receipt.status}`}>
                      {receipt.status}
                    </span>
                  </div>

                  <div className="receipt-compare-meta">
                    <div className="receipt-compare-meta-item">
                      <span className="receipt-compare-label">Date</span>
                      <span>{formatDate(receipt.receipt_date)}</span>
                    </div>
                    {receipt.receipt_time && (
                      <div className="receipt-compare-meta-item">
                        <span className="receipt-compare-label">Time</span>
                        <span>{receipt.receipt_time}</span>
                      </div>
                    )}
                    <div className="receipt-compare-meta-item">
                      <span className="receipt-compare-label">Total</span>
                      <span style={{ fontWeight: 600, fontSize: 16 }}>
                        {receipt.total_amount != null ? `${currencySymbol}${receipt.total_amount.toFixed(2)}` : '-'}
                      </span>
                    </div>
                    {receipt.currency && (
                      <div className="receipt-compare-meta-item">
                        <span className="receipt-compare-label">Currency</span>
                        <span>{receipt.currency}</span>
                      </div>
                    )}
                    {receipt.payment_method && (
                      <div className="receipt-compare-meta-item">
                        <span className="receipt-compare-label">Payment</span>
                        <span>{receipt.payment_method}</span>
                      </div>
                    )}
                  </div>

                  {/* Line items */}
                  {receipt.items.length > 0 && (
                    <div className="receipt-compare-items">
                      <div className="receipt-compare-items-title">Items</div>
                      <table className="table" style={{ fontSize: 13 }}>
                        <thead>
                          <tr>
                            <th className="table-head">Item</th>
                            <th className="table-head right">Qty</th>
                            <th className="table-head right">Price</th>
                            <th className="table-head right">Total</th>
                          </tr>
                        </thead>
                        <tbody>
                          {receipt.items.map((item) => (
                            <tr key={item.id}>
                              <td className="table-cell">{item.name}</td>
                              <td className="table-cell right">{item.quantity ?? '-'}</td>
                              <td className="table-cell right">
                                {item.unit_price != null ? `${currencySymbol}${item.unit_price.toFixed(2)}` : '-'}
                              </td>
                              <td className="table-cell right">
                                {item.line_total != null ? `${currencySymbol}${item.line_total.toFixed(2)}` : '-'}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}

                  {/* Match suggestion */}
                  {receipt.matched_transaction_id && receipt.status !== 'confirmed' && (
                    <div className="receipt-compare-match">
                      Suggested match: {receipt.matched_transaction_description || 'Transaction'} ·{' '}
                      {receipt.matched_transaction_date ? formatDate(receipt.matched_transaction_date) : '-'} ·{' '}
                      {receipt.matched_transaction_amount !== null ? `${currencySymbol}${receipt.matched_transaction_amount?.toFixed(2)}` : '-'}
                      {receipt.matched_reason ? ` (${receipt.matched_reason})` : ''}
                      <button className="btn-secondary" style={{ marginLeft: 8, padding: '2px 10px', fontSize: 12 }} onClick={() => handleLinkMatch(receipt)}>
                        Link match
                      </button>
                    </div>
                  )}

                  {/* Actions */}
                  {receipt.status !== 'confirmed' ? (
                    <div className="receipt-compare-actions">
                      <input
                        type="text"
                        className="filter-input"
                        placeholder="Merchant"
                        defaultValue={receipt.merchant_name || ''}
                        onBlur={(e) => {
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: { ...prev[receipt.id], merchant_name: e.target.value || undefined },
                          }));
                        }}
                      />
                      <input
                        type="date"
                        className="filter-input"
                        defaultValue={receipt.receipt_date || ''}
                        onBlur={(e) => {
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: { ...prev[receipt.id], receipt_date: e.target.value || undefined },
                          }));
                        }}
                      />
                      <input
                        type="number"
                        step="0.01"
                        className="filter-input"
                        placeholder="Total"
                        defaultValue={receipt.total_amount ?? ''}
                        onBlur={(e) => {
                          const value = e.target.value ? Number(e.target.value) : undefined;
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: { ...prev[receipt.id], total_amount: value },
                          }));
                        }}
                      />
                      <select
                        className="filter-input"
                        defaultValue=""
                        onChange={(e) => {
                          const value = e.target.value ? Number(e.target.value) : undefined;
                          setOverrides((prev) => ({
                            ...prev,
                            [receipt.id]: { ...prev[receipt.id], category_id: value },
                          }));
                        }}
                      >
                        <option value="">Category</option>
                        {categories.filter(c => c.is_expense).map((cat) => (
                          <option key={cat.id} value={cat.id}>{cat.name}</option>
                        ))}
                      </select>
                      <button className="btn-primary" onClick={() => handleConfirm(receipt)}>
                        Confirm
                      </button>
                    </div>
                  ) : (
                    <div className="receipt-compare-confirmed">
                      Linked to transaction #{receipt.transaction_id}
                    </div>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        {totalPages > 1 && (
          <div className="pager" style={{ marginTop: 16 }}>
            <button className="pager-btn" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
              Previous
            </button>
            <span className="pager-label">Page {page} of {totalPages}</span>
            <button className="pager-btn" disabled={page === totalPages} onClick={() => setPage((p) => p + 1)}>
              Next
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
