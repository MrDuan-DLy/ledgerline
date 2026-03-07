import { useState, useEffect, Fragment } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  getTransactions,
  getCategories,
  getSummary,
  updateTransaction,
  deleteTransaction,
  bulkClassify,
  bulkDelete,
  bulkExclude,
  getReceiptByTransaction,
  updateReceiptItem,
  Transaction,
  Category,
  Summary,
  Receipt,
} from '../api/client';
import { formatExpense, formatDate, formatCurrency } from '../utils/format';
import { useAppConfig } from '../contexts/AppConfig';

export default function Transactions() {
  const { currencySymbol } = useAppConfig();
  const [searchParams, setSearchParams] = useSearchParams();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [unclassifiedOnly, setUnclassifiedOnly] = useState(false);
  const [search, setSearch] = useState('');
  const [showExcluded, setShowExcluded] = useState(true);
  const [expandedIds, setExpandedIds] = useState<Set<number>>(new Set());
  const [receiptMap, setReceiptMap] = useState<Record<number, Receipt | null | undefined>>({});
  const [editTxnId, setEditTxnId] = useState<number | null>(null);
  const [txnDrafts, setTxnDrafts] = useState<Record<number, { date: string; description: string }>>({});
  const [editingItemId, setEditingItemId] = useState<number | null>(null);
  const [itemDrafts, setItemDrafts] = useState<Record<number, { name: string; line_total: string }>>({});

  const statementId = searchParams.get('statement_id') ? Number(searchParams.get('statement_id')) : undefined;

  const clearStatementFilter = () => {
    searchParams.delete('statement_id');
    setSearchParams(searchParams);
    setPage(1);
  };

  const loadData = async () => {
    setLoading(true);
    try {
      const [txnRes, catRes, sumRes] = await Promise.all([
        getTransactions({ page, page_size: 50, unclassified_only: unclassifiedOnly, hide_excluded: !showExcluded, search, statement_id: statementId }),
        getCategories(),
        getSummary(),
      ]);
      setTransactions(txnRes.items);
      setTotalPages(txnRes.total_pages);
      setCategories(catRes);
      setSummary(sumRes);
    } catch (e) {
      console.error(e);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadData();
  }, [page, unclassifiedOnly, showExcluded, search, statementId]);

  const handleCategoryChange = async (txnId: number, categoryId: number) => {
    await updateTransaction(txnId, { category_id: categoryId });
    loadData();
  };

  const handleDelete = async (txnId: number) => {
    if (!confirm('Delete this transaction?')) return;
    await deleteTransaction(txnId);
    loadData();
  };

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return;
    if (!confirm(`Delete ${selectedIds.size} selected transaction(s)?`)) return;
    await bulkDelete(Array.from(selectedIds));
    setSelectedIds(new Set());
    loadData();
  };

  const handleBulkExclude = async (exclude: boolean) => {
    if (selectedIds.size === 0) return;
    await bulkExclude(Array.from(selectedIds), exclude);
    setSelectedIds(new Set());
    loadData();
  };

  const handleBulkClassify = async (categoryId: number) => {
    if (selectedIds.size === 0) return;
    await bulkClassify(Array.from(selectedIds), categoryId);
    setSelectedIds(new Set());
    loadData();
  };

  const toggleSelect = (id: number) => {
    const newSet = new Set(selectedIds);
    if (newSet.has(id)) {
      newSet.delete(id);
    } else {
      newSet.add(id);
    }
    setSelectedIds(newSet);
  };

  const toggleExpand = async (txnId: number) => {
    const next = new Set(expandedIds);
    if (next.has(txnId)) {
      next.delete(txnId);
      setExpandedIds(next);
      return;
    }
    next.add(txnId);
    setExpandedIds(next);
    if (!(txnId in receiptMap)) {
      const receipt = await getReceiptByTransaction(txnId);
      setReceiptMap((prev) => ({ ...prev, [txnId]: receipt }));
    }
  };

  const startEditTxn = (txn: Transaction) => {
    setEditTxnId(txn.id);
    setTxnDrafts((prev) => ({
      ...prev,
      [txn.id]: {
        date: txn.effective_date || txn.raw_date,
        description: txn.description || txn.raw_description,
      },
    }));
  };

  const saveTxn = async (txn: Transaction) => {
    const draft = txnDrafts[txn.id];
    if (!draft) return;
    await updateTransaction(txn.id, {
      effective_date: draft.date,
      description: draft.description,
    });
    setEditTxnId(null);
    loadData();
  };

  const cancelEditTxn = () => {
    setEditTxnId(null);
  };

  const refreshReceipt = async (txnId: number) => {
    const receipt = await getReceiptByTransaction(txnId);
    setReceiptMap((prev) => ({ ...prev, [txnId]: receipt }));
  };

  const selectAll = () => {
    if (selectedIds.size === transactions.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(transactions.map(t => t.id)));
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Transactions</p>
          <h1>Review, classify, and keep it tidy.</h1>
        </div>
        <div className="page-note">Changes save instantly. Manual tags override rules.</div>
      </div>

      {statementId && (
        <div className="panel" style={{ padding: '10px 16px', display: 'flex', alignItems: 'center', gap: 10, fontSize: 14 }}>
          <span>Showing transactions from statement #{statementId}</span>
          <button className="btn-secondary" style={{ padding: '2px 10px', fontSize: 12 }} onClick={clearStatementFilter}>
            Show all
          </button>
        </div>
      )}

      {/* Summary Cards */}
      {summary && (
        <div className="kpi-grid tight">
          <div className="panel kpi-card">
            <div className="kpi-label">Total Transactions</div>
            <div className="kpi-value">{summary.total_transactions}</div>
            <div className="kpi-meta">All time</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Total Expenses</div>
            <div className="kpi-value">{currencySymbol}{summary.total_expenses.toFixed(2)}</div>
            <div className="kpi-meta">All spending</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Unclassified</div>
            <div className="kpi-value">{summary.unclassified_count}</div>
            <div className="kpi-meta">Needs review</div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="panel filter-bar">
        <input
          type="text"
          placeholder="Search description..."
          className="filter-input"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
        />
        <label className="filter-check">
          <input
            type="checkbox"
            checked={unclassifiedOnly}
            onChange={(e) => { setUnclassifiedOnly(e.target.checked); setPage(1); }}
          />
          Unclassified only
        </label>
        <label className="filter-check">
          <input
            type="checkbox"
            checked={!showExcluded}
            onChange={(e) => { setShowExcluded(!e.target.checked); setPage(1); }}
          />
          Hide excluded
        </label>
      </div>

      {/* Bulk Actions */}
      {selectedIds.size > 0 && (
        <div className="panel bulk-bar">
          <span className="bulk-count">{selectedIds.size} selected</span>
          <select
            className="bulk-select"
            onChange={(e) => handleBulkClassify(parseInt(e.target.value))}
            defaultValue=""
          >
            <option value="" disabled>Classify as...</option>
            {categories.filter(c => c.is_expense).map(c => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <button
            className="bulk-clear"
            onClick={() => handleBulkExclude(true)}
          >
            Exclude from stats
          </button>
          <button
            className="bulk-clear"
            onClick={() => handleBulkExclude(false)}
          >
            Include in stats
          </button>
          <button
            className="bulk-delete"
            onClick={handleBulkDelete}
          >
            Delete selected
          </button>
          <button
            className="bulk-clear"
            onClick={() => setSelectedIds(new Set())}
          >
            Clear selection
          </button>
        </div>
      )}

      {/* Table */}
      <div className="panel table-card">
        <table className="table">
          <thead>
            <tr>
              <th className="table-head">
                <input
                  type="checkbox"
                  checked={selectedIds.size === transactions.length && transactions.length > 0}
                  onChange={selectAll}
                />
              </th>
              <th className="table-head">Date</th>
              <th className="table-head">Description</th>
              <th className="table-head right">Amount</th>
              <th className="table-head">Category</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="table-empty">Loading...</td></tr>
            ) : transactions.length === 0 ? (
              <tr><td colSpan={5} className="table-empty">No transactions found</td></tr>
            ) : (
              transactions.map((txn) => (
                <Fragment key={txn.id}>
                  <tr className={`${selectedIds.has(txn.id) ? 'selected' : ''}${txn.is_excluded ? ' excluded' : ''}`}>
                    <td className="table-cell">
                      <input
                        type="checkbox"
                        checked={selectedIds.has(txn.id)}
                        onChange={() => toggleSelect(txn.id)}
                      />
                    </td>
                    <td className="table-cell">
                      {editTxnId === txn.id ? (
                        <input
                          type="date"
                          className="table-select"
                          value={txnDrafts[txn.id]?.date || ''}
                          onChange={(e) =>
                            setTxnDrafts((prev) => ({
                              ...prev,
                              [txn.id]: { ...prev[txn.id], date: e.target.value },
                            }))
                          }
                        />
                      ) : (
                        formatDate(txn.effective_date || txn.raw_date)
                      )}
                    </td>
                    <td className="table-cell">
                      <div className="txn-desc">
                        {editTxnId === txn.id ? (
                          <input
                            type="text"
                            className="table-select"
                            value={txnDrafts[txn.id]?.description || ''}
                            onChange={(e) =>
                              setTxnDrafts((prev) => ({
                                ...prev,
                                [txn.id]: { ...prev[txn.id], description: e.target.value },
                              }))
                            }
                          />
                        ) : (
                          <span>{txn.description || txn.raw_description}</span>
                        )}
                        <div className="txn-actions">
                          {editTxnId === txn.id ? (
                            <>
                              <button className="link-button" onClick={() => saveTxn(txn)}>
                                Save
                              </button>
                              <button className="link-button" onClick={cancelEditTxn}>
                                Cancel
                              </button>
                            </>
                          ) : (
                            <>
                              <button className="link-button" onClick={() => startEditTxn(txn)}>
                                Edit
                              </button>
                              <button className="link-button" onClick={() => toggleExpand(txn.id)}>
                                {expandedIds.has(txn.id) ? 'Hide' : 'Details'}
                              </button>
                              <button
                                className="link-button"
                                onClick={async () => {
                                  await bulkExclude([txn.id], !txn.is_excluded);
                                  loadData();
                                }}
                              >
                                {txn.is_excluded ? 'Include' : 'Exclude'}
                              </button>
                              <button className="link-button link-danger" onClick={() => handleDelete(txn.id)}>
                                Delete
                              </button>
                            </>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className={`table-cell right amount${txn.amount > 0 ? ' pos' : ''}`}>
                      {formatExpense(txn.amount, currencySymbol)}
                    </td>
                    <td className="table-cell">
                      <select
                        className={`table-select ${!txn.category_id ? 'unassigned' : ''}`}
                        value={txn.category_id || ''}
                        onChange={(e) => handleCategoryChange(txn.id, parseInt(e.target.value))}
                      >
                        <option value="">-- Select --</option>
                        {categories.map(c => (
                          <option key={c.id} value={c.id}>{c.name}</option>
                        ))}
                      </select>
                    </td>
                  </tr>
                  {expandedIds.has(txn.id) && (
                    <tr className="details-row">
                      <td className="table-cell" />
                      <td className="table-cell" colSpan={4}>
                        <div className="receipt-details">
                          {receiptMap[txn.id] === undefined && (
                            <div className="receipt-empty">Loading receipt...</div>
                          )}
                          {receiptMap[txn.id] === null && (
                            <div className="receipt-empty">No receipt linked to this transaction.</div>
                          )}
                          {receiptMap[txn.id] && (
                            <>
                              <div className="receipt-header">
                                <div>{receiptMap[txn.id]?.merchant_name || 'Receipt'}</div>
                                <div>
                                  {receiptMap[txn.id]?.total_amount !== null
                                    ? `${currencySymbol}${receiptMap[txn.id]?.total_amount?.toFixed(2)}`
                                    : '-'}
                                </div>
                              </div>
                              <div className="receipt-items">
                                {receiptMap[txn.id]?.items.length ? (
                                  receiptMap[txn.id]?.items.map((item) => (
                                    <div key={item.id} className="receipt-item">
                                      {editingItemId === item.id ? (
                                        <>
                                          <input
                                            type="text"
                                            className="table-select"
                                            value={itemDrafts[item.id]?.name || item.name}
                                            onChange={(e) =>
                                              setItemDrafts((prev) => ({
                                                ...prev,
                                                [item.id]: {
                                                  name: e.target.value,
                                                  line_total: prev[item.id]?.line_total || '',
                                                },
                                              }))
                                            }
                                          />
                                          <input
                                            type="number"
                                            step="0.01"
                                            className="table-select"
                                            value={itemDrafts[item.id]?.line_total ?? ''}
                                            onChange={(e) =>
                                              setItemDrafts((prev) => ({
                                                ...prev,
                                                [item.id]: {
                                                  name: prev[item.id]?.name || item.name,
                                                  line_total: e.target.value,
                                                },
                                              }))
                                            }
                                          />
                                          <button
                                            className="link-button"
                                            onClick={async () => {
                                              const draft = itemDrafts[item.id];
                                              const lineTotal =
                                                draft?.line_total !== undefined
                                                  ? Number(draft.line_total)
                                                  : item.line_total;
                                              await updateReceiptItem(item.id, {
                                                name: draft?.name || item.name,
                                                line_total: lineTotal,
                                              });
                                              setEditingItemId(null);
                                              refreshReceipt(txn.id);
                                            }}
                                          >
                                            Save
                                          </button>
                                          <button
                                            className="link-button"
                                            onClick={() => setEditingItemId(null)}
                                          >
                                            Cancel
                                          </button>
                                        </>
                                      ) : (
                                        <>
                                          <span>{item.name}</span>
                                          <span>
                                            {item.line_total !== null ? `${currencySymbol}${item.line_total?.toFixed(2)}` : ''}
                                          </span>
                                          <button
                                            className="link-button"
                                            onClick={() => {
                                              setEditingItemId(item.id);
                                              setItemDrafts((prev) => ({
                                                ...prev,
                                                [item.id]: {
                                                  name: item.name,
                                                  line_total: item.line_total !== null ? item.line_total.toFixed(2) : '',
                                                },
                                              }));
                                            }}
                                          >
                                            Edit
                                          </button>
                                        </>
                                      )}
                                    </div>
                                  ))
                                ) : (
                                  <div className="receipt-empty">No item details found.</div>
                                )}
                              </div>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="pager">
          <button
            className="pager-btn"
            disabled={page === 1}
            onClick={() => setPage(p => p - 1)}
          >
            Previous
          </button>
          <span className="pager-label">Page {page} of {totalPages}</span>
          <button
            className="pager-btn"
            disabled={page === totalPages}
            onClick={() => setPage(p => p + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
