import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  getImportSession,
  updateImportItem,
  confirmImportSession,
  deleteImportSession,
  getCategories,
  importSourceUrl,
  importPageUrl,
  ImportSession,
  ImportItem,
  Category,
} from '../api/client';

interface ReceiptLineItem {
  name: string;
  quantity: number | null;
  unit_price: number | null;
  line_total: number | null;
}

interface AiUsage {
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  duration_ms: number;
}

export default function Review() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();

  const [session, setSession] = useState<ImportSession | null>(null);
  const [, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [confirming, setConfirming] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [drafts, setDrafts] = useState<Record<number, Partial<ImportItem>>>({});

  useEffect(() => {
    if (!sessionId) return;
    Promise.all([
      getImportSession(sessionId),
      getCategories(),
    ]).then(([sess, cats]) => {
      setSession(sess);
      setCategories(cats);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [sessionId]);

  if (loading) {
    return (
      <div className="page">
        <div className="panel" style={{ padding: 40, textAlign: 'center' }}>
          <div className="spinner" />
          <p>Loading review session...</p>
        </div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="page">
        <div className="panel" style={{ padding: 40, textAlign: 'center' }}>
          <p>Import session not found.</p>
          <button className="btn-primary" style={{ marginTop: 16 }} onClick={() => navigate('/import')}>
            Back to Import
          </button>
        </div>
      </div>
    );
  }

  const isReadonly = session.status !== 'pending';

  const items = session.items;
  const pendingItems = items.filter(i => i.status === 'pending' || i.status === 'confirmed');
  const skippedItems = items.filter(i => i.status === 'skipped');
  const duplicates = items.filter(i => i.duplicate_of_id !== null);

  const isPdf = session.source_type === 'pdf';
  const isReceipt = session.source_type === 'receipt_image';
  const pageCount = session.page_count || 0;
  const pagePaths = session.page_image_paths || [];

  const metadata = session.metadata_json ? JSON.parse(session.metadata_json) : null;
  const aiUsage: AiUsage | null = session.ai_usage_json ? JSON.parse(session.ai_usage_json) : null;

  // For receipt: parse line items from first item
  const receiptItem = isReceipt && items.length > 0 ? items[0] : null;
  const receiptLineItems: ReceiptLineItem[] = receiptItem?.extracted_items_json
    ? JSON.parse(receiptItem.extracted_items_json)
    : [];

  const toggleSkip = async (item: ImportItem) => {
    if (!sessionId) return;
    const newStatus = item.status === 'skipped' ? 'pending' : 'skipped';
    const updated = await updateImportItem(sessionId, item.id, { status: newStatus });
    setSession(prev => prev ? {
      ...prev,
      items: prev.items.map(i => i.id === item.id ? { ...i, ...updated } : i),
    } : prev);
  };

  const startEdit = (item: ImportItem) => {
    setEditingId(item.id);
    setDrafts(prev => ({
      ...prev,
      [item.id]: {
        extracted_date: item.extracted_date,
        extracted_description: item.extracted_description,
        extracted_amount: item.extracted_amount,
      },
    }));
  };

  const saveEdit = async (item: ImportItem) => {
    if (!sessionId) return;
    const draft = drafts[item.id];
    if (!draft) return;
    const updated = await updateImportItem(sessionId, item.id, {
      extracted_date: draft.extracted_date || undefined,
      extracted_description: draft.extracted_description || undefined,
      extracted_amount: draft.extracted_amount ?? undefined,
    });
    setSession(prev => prev ? {
      ...prev,
      items: prev.items.map(i => i.id === item.id ? { ...i, ...updated } : i),
    } : prev);
    setEditingId(null);
  };

  const unlinkDuplicate = async (item: ImportItem) => {
    if (!sessionId) return;
    const updated = await updateImportItem(sessionId, item.id, {
      status: 'confirmed',
      duplicate_of_id: null,
    });
    setSession(prev => prev ? {
      ...prev,
      items: prev.items.map(i => i.id === item.id ? { ...i, ...updated, duplicate_of_id: null, duplicate_reason: null, duplicate_score: null } : i),
    } : prev);
  };

  const handleConfirm = async () => {
    if (!sessionId) return;
    setConfirming(true);
    try {
      const result = await confirmImportSession(sessionId);
      if (result.success) {
        navigate('/transactions');
      }
    } catch {
      // stay on page
    }
    setConfirming(false);
  };

  const handleDiscard = async () => {
    if (!sessionId) return;
    await deleteImportSession(sessionId);
    navigate('/import');
  };

  const goToPage = (item: ImportItem) => {
    if (item.page_num) {
      setCurrentPage(item.page_num);
    }
  };

  const formatDate = (d: string | null) => {
    if (!d) return '-';
    return new Date(d + 'T00:00:00').toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: '2-digit' });
  };

  const formatAmount = (a: number | null) => {
    if (a === null) return '-';
    if (a > 0) return `+\u00A3${a.toFixed(2)}`;
    return `\u00A3${Math.abs(a).toFixed(2)}`;
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">{isReadonly ? 'Import Details' : 'Review Import'}</p>
          <h1>{session.source_file}</h1>
        </div>
        {isReadonly ? (
          <button className="btn-secondary" onClick={() => navigate('/import')}>Back to Import</button>
        ) : (
          <div style={{ display: 'flex', gap: 10 }}>
            <button className="btn-secondary" onClick={handleDiscard}>Discard</button>
            <button className="btn-primary" onClick={handleConfirm} disabled={confirming}>
              {confirming ? 'Confirming...' : `Confirm ${pendingItems.length} items`}
            </button>
          </div>
        )}
      </div>

      {/* AI Usage Stats */}
      {aiUsage && (
        <div className="panel ai-usage-bar">
          <span className="ai-usage-model">{aiUsage.model}</span>
          <span className="ai-usage-stat">
            <span className="ai-usage-label">Prompt</span>
            <span className="ai-usage-value">{aiUsage.prompt_tokens.toLocaleString()}</span>
          </span>
          <span className="ai-usage-stat">
            <span className="ai-usage-label">Completion</span>
            <span className="ai-usage-value">{aiUsage.completion_tokens.toLocaleString()}</span>
          </span>
          <span className="ai-usage-stat">
            <span className="ai-usage-label">Total</span>
            <span className="ai-usage-value">{aiUsage.total_tokens.toLocaleString()} tokens</span>
          </span>
          <span className="ai-usage-stat">
            <span className="ai-usage-label">Time</span>
            <span className="ai-usage-value">{formatDuration(aiUsage.duration_ms)}</span>
          </span>
        </div>
      )}

      {/* PDF metadata KPIs */}
      {isPdf && metadata && (
        <div className="kpi-grid">
          <div className="panel kpi-card">
            <div className="kpi-label">Period</div>
            <div className="kpi-value">{formatDate(metadata.period_start)} - {formatDate(metadata.period_end)}</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Opening Balance</div>
            <div className="kpi-value">{metadata.opening_balance != null ? `\u00A3${metadata.opening_balance.toFixed(2)}` : '-'}</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Closing Balance</div>
            <div className="kpi-value">{metadata.closing_balance != null ? `\u00A3${metadata.closing_balance.toFixed(2)}` : '-'}</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Transactions</div>
            <div className="kpi-value">{items.length}</div>
            <div className="kpi-meta">{duplicates.length} potential duplicates</div>
          </div>
        </div>
      )}

      <div className="review-layout">
        {/* Left: Source Preview */}
        <div className="review-source panel">
          {isPdf && pagePaths.length > 0 ? (
            <>
              <div className="review-page-image">
                <img
                  src={importPageUrl(session.id, currentPage)}
                  alt={`Page ${currentPage}`}
                  style={{ width: '100%', borderRadius: 8 }}
                />
              </div>
              {pageCount > 1 && (
                <div className="review-page-nav">
                  <button
                    className="btn-secondary"
                    disabled={currentPage <= 1}
                    onClick={() => setCurrentPage(p => p - 1)}
                  >
                    &larr; Prev
                  </button>
                  <span className="pager-label">Page {currentPage} of {pageCount}</span>
                  <button
                    className="btn-secondary"
                    disabled={currentPage >= pageCount}
                    onClick={() => setCurrentPage(p => p + 1)}
                  >
                    Next &rarr;
                  </button>
                </div>
              )}
            </>
          ) : isReceipt ? (
            <div className="review-page-image">
              <img
                src={importSourceUrl(session.id)}
                alt="Receipt"
                style={{ width: '100%', borderRadius: 8 }}
              />
            </div>
          ) : (
            <div style={{ padding: 40, textAlign: 'center', color: 'var(--muted)' }}>
              No preview available
            </div>
          )}
        </div>

        {/* Right: Extracted Data */}
        <div className="review-items">
          {/* Receipt-specific detail card */}
          {isReceipt && receiptItem && metadata && (
            <div className="panel review-receipt-detail">
              <div className="review-receipt-header">
                <h2 className="review-receipt-merchant">{metadata.merchant_name || receiptItem.extracted_merchant || 'Unknown Merchant'}</h2>
                <span className={`review-item-amount ${(receiptItem.extracted_amount ?? 0) < 0 ? 'neg' : 'pos'}`} style={{ fontSize: 22 }}>
                  {formatAmount(receiptItem.extracted_amount)}
                </span>
              </div>

              <div className="review-receipt-meta-grid">
                <div className="review-receipt-meta-item">
                  <span className="review-receipt-meta-label">Date</span>
                  <span>{formatDate(receiptItem.extracted_date)}</span>
                </div>
                {metadata.receipt_time && (
                  <div className="review-receipt-meta-item">
                    <span className="review-receipt-meta-label">Time</span>
                    <span>{metadata.receipt_time}</span>
                  </div>
                )}
                {metadata.currency && (
                  <div className="review-receipt-meta-item">
                    <span className="review-receipt-meta-label">Currency</span>
                    <span>{metadata.currency}</span>
                  </div>
                )}
                {metadata.payment_method && (
                  <div className="review-receipt-meta-item">
                    <span className="review-receipt-meta-label">Payment</span>
                    <span>{metadata.payment_method}</span>
                  </div>
                )}
              </div>

              {/* Line items table */}
              {receiptLineItems.length > 0 && (
                <div className="review-receipt-items">
                  <div className="review-receipt-items-header">Items</div>
                  <table className="table">
                    <thead>
                      <tr>
                        <th className="table-head">Item</th>
                        <th className="table-head right">Qty</th>
                        <th className="table-head right">Price</th>
                        <th className="table-head right">Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      {receiptLineItems.map((li, idx) => (
                        <tr key={idx}>
                          <td className="table-cell">{li.name}</td>
                          <td className="table-cell right">{li.quantity ?? '-'}</td>
                          <td className="table-cell right">{li.unit_price != null ? `\u00A3${li.unit_price.toFixed(2)}` : '-'}</td>
                          <td className="table-cell right">{li.line_total != null ? `\u00A3${li.line_total.toFixed(2)}` : '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {/* Duplicate warning for receipt */}
              {receiptItem.duplicate_of_id !== null && (
                <div className="review-dup-warning">
                  <span className="review-dup-icon">&#9888;</span>
                  <span>
                    Potential duplicate: {receiptItem.duplicate_reason}
                    {receiptItem.duplicate_transaction_description && (
                      <> &mdash; &ldquo;{receiptItem.duplicate_transaction_description}&rdquo; ({formatAmount(receiptItem.duplicate_transaction_amount)})</>
                    )}
                  </span>
                  {!isReadonly && (
                    <>
                      <button className="link-button" onClick={() => toggleSkip(receiptItem)}>Skip</button>
                      <button className="link-button" onClick={() => unlinkDuplicate(receiptItem)}>Import anyway</button>
                    </>
                  )}
                </div>
              )}

              {/* Action row */}
              {isReadonly ? (
                <div className="review-receipt-actions">
                  <span className={`panel-pill ${receiptItem.status === 'skipped' ? '' : 'pill-green'}`}>
                    {receiptItem.status === 'skipped' ? 'Skipped' : 'Imported'}
                  </span>
                </div>
              ) : (
                <>
                  <div className="review-receipt-actions">
                    <label className="review-item-check">
                      <input
                        type="checkbox"
                        checked={receiptItem.status !== 'skipped'}
                        onChange={() => toggleSkip(receiptItem)}
                      />
                      <span style={{ marginLeft: 6 }}>
                        {receiptItem.status === 'skipped' ? 'Skipped' : 'Include in import'}
                      </span>
                    </label>
                    <button className="link-button" onClick={() => startEdit(receiptItem)}>Edit details</button>
                  </div>

                  {/* Inline edit for receipt */}
                  {editingId === receiptItem.id && (
                    <div className="review-item-edit" style={{ marginTop: 8 }}>
                      <input
                        type="date"
                        value={drafts[receiptItem.id]?.extracted_date || ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [receiptItem.id]: { ...prev[receiptItem.id], extracted_date: e.target.value } }))}
                        className="filter-input"
                        style={{ width: 140, padding: '4px 8px' }}
                      />
                      <input
                        type="text"
                        value={drafts[receiptItem.id]?.extracted_description || ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [receiptItem.id]: { ...prev[receiptItem.id], extracted_description: e.target.value } }))}
                        className="filter-input"
                        style={{ flex: 1, padding: '4px 8px' }}
                        placeholder="Description"
                      />
                      <input
                        type="number"
                        step="0.01"
                        value={drafts[receiptItem.id]?.extracted_amount ?? ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [receiptItem.id]: { ...prev[receiptItem.id], extracted_amount: parseFloat(e.target.value) || 0 } }))}
                        className="filter-input"
                        style={{ width: 100, padding: '4px 8px' }}
                        placeholder="Amount"
                      />
                      <button className="btn-primary" style={{ padding: '4px 12px' }} onClick={() => saveEdit(receiptItem)}>Save</button>
                      <button className="btn-secondary" style={{ padding: '4px 12px' }} onClick={() => setEditingId(null)}>Cancel</button>
                    </div>
                  )}
                </>
              )}
            </div>
          )}

          {/* PDF transaction list */}
          {isPdf && items.map((item) => {
            const isSkipped = item.status === 'skipped';
            const isDup = item.duplicate_of_id !== null;
            const isEditing = !isReadonly && editingId === item.id;
            const draft = drafts[item.id];

            return (
              <div
                key={item.id}
                className={`panel review-item ${isSkipped ? 'review-item-skipped' : ''} ${isDup && !isReadonly ? 'review-item-dup' : ''}`}
                onClick={() => goToPage(item)}
              >
                <div className="review-item-header">
                  {!isReadonly && (
                    <label className="review-item-check" onClick={e => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        checked={!isSkipped}
                        onChange={() => toggleSkip(item)}
                      />
                    </label>
                  )}

                  {isEditing ? (
                    <div className="review-item-edit" onClick={e => e.stopPropagation()}>
                      <input
                        type="date"
                        value={draft?.extracted_date || ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [item.id]: { ...prev[item.id], extracted_date: e.target.value } }))}
                        className="filter-input"
                        style={{ width: 140, padding: '4px 8px' }}
                      />
                      <input
                        type="text"
                        value={draft?.extracted_description || ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [item.id]: { ...prev[item.id], extracted_description: e.target.value } }))}
                        className="filter-input"
                        style={{ flex: 1, padding: '4px 8px' }}
                      />
                      <input
                        type="number"
                        step="0.01"
                        value={draft?.extracted_amount ?? ''}
                        onChange={e => setDrafts(prev => ({ ...prev, [item.id]: { ...prev[item.id], extracted_amount: parseFloat(e.target.value) || 0 } }))}
                        className="filter-input"
                        style={{ width: 100, padding: '4px 8px' }}
                      />
                      <button className="btn-primary" style={{ padding: '4px 12px' }} onClick={() => saveEdit(item)}>Save</button>
                      <button className="btn-secondary" style={{ padding: '4px 12px' }} onClick={() => setEditingId(null)}>Cancel</button>
                    </div>
                  ) : (
                    <>
                      <span className="review-item-date">{formatDate(item.extracted_date)}</span>
                      <span className="review-item-desc">{item.extracted_description || '-'}</span>
                      <span className={`review-item-amount ${(item.extracted_amount ?? 0) < 0 ? 'neg' : 'pos'}`}>
                        {formatAmount(item.extracted_amount)}
                      </span>
                      {item.extracted_balance !== null && (
                        <span className="review-item-balance">bal: {'\u00A3'}{item.extracted_balance?.toFixed(2)}</span>
                      )}
                      {isReadonly && isSkipped && (
                        <span className="panel-pill" style={{ fontSize: 11 }}>Skipped</span>
                      )}
                      {!isReadonly && (
                        <button
                          className="link-button"
                          onClick={e => { e.stopPropagation(); startEdit(item); }}
                        >
                          Edit
                        </button>
                      )}
                    </>
                  )}
                </div>

                {isDup && !isSkipped && !isReadonly && (
                  <div className="review-dup-warning" onClick={e => e.stopPropagation()}>
                    <span className="review-dup-icon">&#9888;</span>
                    <span>
                      Potential duplicate: {item.duplicate_reason}
                      {item.duplicate_transaction_description && (
                        <> &mdash; &ldquo;{item.duplicate_transaction_description}&rdquo; ({formatAmount(item.duplicate_transaction_amount)})</>
                      )}
                    </span>
                    <button className="link-button" onClick={() => toggleSkip(item)}>Skip</button>
                    <button className="link-button" onClick={() => unlinkDuplicate(item)}>Import anyway</button>
                  </div>
                )}

                {item.page_num && isPdf && (
                  <span className="review-item-page">p.{item.page_num}</span>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Summary footer */}
      <div className="panel review-footer">
        <span>{items.length} items total</span>
        <span className="review-footer-sep">|</span>
        <span>{pendingItems.length} imported</span>
        <span className="review-footer-sep">|</span>
        <span>{skippedItems.length} skipped</span>
        {duplicates.length > 0 && (
          <>
            <span className="review-footer-sep">|</span>
            <span>{duplicates.length} duplicates</span>
          </>
        )}
        <div style={{ flex: 1 }} />
        {isReadonly ? (
          <button className="btn-secondary" onClick={() => navigate('/import')}>Back to Import</button>
        ) : (
          <>
            <button className="btn-secondary" onClick={handleDiscard}>Discard</button>
            <button className="btn-primary" onClick={handleConfirm} disabled={confirming}>
              {confirming ? 'Confirming...' : 'Confirm Selected'}
            </button>
          </>
        )}
      </div>
    </div>
  );
}
