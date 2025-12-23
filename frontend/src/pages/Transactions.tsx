import { useState, useEffect } from 'react';
import {
  getTransactions,
  getCategories,
  getSummary,
  updateTransaction,
  bulkClassify,
  Transaction,
  Category,
  Summary,
} from '../api/client';

export default function Transactions() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [unclassifiedOnly, setUnclassifiedOnly] = useState(false);
  const [search, setSearch] = useState('');

  const loadData = async () => {
    setLoading(true);
    try {
      const [txnRes, catRes, sumRes] = await Promise.all([
        getTransactions({ page, page_size: 50, unclassified_only: unclassifiedOnly, search }),
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
  }, [page, unclassifiedOnly, search]);

  const handleCategoryChange = async (txnId: number, categoryId: number) => {
    await updateTransaction(txnId, { category_id: categoryId });
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

  const selectAll = () => {
    if (selectedIds.size === transactions.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(transactions.map(t => t.id)));
    }
  };

  const formatAmount = (amount: number) => {
    const abs = Math.abs(amount).toFixed(2);
    return amount < 0 ? `-£${abs}` : `£${abs}`;
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
          <p className="eyebrow">Transactions</p>
          <h1>Review, classify, and keep it tidy.</h1>
        </div>
        <div className="page-note">Changes save instantly. Manual tags override rules.</div>
      </div>

      {/* Summary Cards */}
      {summary && (
        <div className="kpi-grid tight">
          <div className="panel kpi-card">
            <div className="kpi-label">Total Transactions</div>
            <div className="kpi-value">{summary.total_transactions}</div>
            <div className="kpi-meta">All time</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Income</div>
            <div className="kpi-value">£{summary.total_income.toFixed(2)}</div>
            <div className="kpi-meta">Positive cashflow</div>
          </div>
          <div className="panel kpi-card">
            <div className="kpi-label">Expenses</div>
            <div className="kpi-value">£{summary.total_expenses.toFixed(2)}</div>
            <div className="kpi-meta">Outflow</div>
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
                <tr key={txn.id} className={selectedIds.has(txn.id) ? 'selected' : ''}>
                  <td className="table-cell">
                    <input
                      type="checkbox"
                      checked={selectedIds.has(txn.id)}
                      onChange={() => toggleSelect(txn.id)}
                    />
                  </td>
                  <td className="table-cell">{formatDate(txn.raw_date)}</td>
                  <td className="table-cell">{txn.raw_description}</td>
                  <td className={`table-cell right amount ${txn.amount < 0 ? 'neg' : 'pos'}`}>
                    {formatAmount(txn.amount)}
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
