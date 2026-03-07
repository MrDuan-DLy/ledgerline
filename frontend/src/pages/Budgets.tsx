import { useState, useEffect } from 'react';
import {
  getBudgets,
  getBudgetStatus,
  createBudget,
  updateBudget,
  deleteBudget,
  getCategories,
  BudgetResponse,
  BudgetStatusResponse,
  Category,
} from '../api/client';
import { toast } from '../components/Toast';
import { useAppConfig } from '../contexts/AppConfig';

export default function Budgets() {
  const { currencySymbol } = useAppConfig();
  const [budgets, setBudgets] = useState<BudgetResponse[]>([]);
  const [status, setStatus] = useState<BudgetStatusResponse | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);

  const [newCategoryId, setNewCategoryId] = useState<number | ''>('');
  const [newLimit, setNewLimit] = useState('');
  const [editId, setEditId] = useState<number | null>(null);
  const [editLimit, setEditLimit] = useState('');

  const load = async () => {
    setLoading(true);
    try {
      const [b, s, c] = await Promise.all([
        getBudgets(),
        getBudgetStatus(),
        getCategories(),
      ]);
      setBudgets(b);
      setStatus(s);
      setCategories(c);
    } catch (e: any) {
      toast(e.message, 'error');
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const budgetCategoryIds = new Set(budgets.map((b) => b.category_id));
  const availableCategories = categories.filter(
    (c) => c.is_expense && !budgetCategoryIds.has(c.id),
  );

  const handleCreate = async () => {
    if (!newCategoryId || !newLimit) return;
    try {
      await createBudget({ category_id: Number(newCategoryId), monthly_limit: Number(newLimit) });
      setNewCategoryId('');
      setNewLimit('');
      load();
      toast('Budget created', 'success');
    } catch (e: any) {
      toast(e.message, 'error');
    }
  };

  const handleUpdate = async (id: number) => {
    if (!editLimit) return;
    try {
      await updateBudget(id, { monthly_limit: Number(editLimit) });
      setEditId(null);
      load();
      toast('Budget updated', 'success');
    } catch (e: any) {
      toast(e.message, 'error');
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Remove this budget?')) return;
    try {
      await deleteBudget(id);
      load();
      toast('Budget removed', 'success');
    } catch (e: any) {
      toast(e.message, 'error');
    }
  };

  const statusMap = new Map(status?.items.map((s) => [s.category_id, s]) ?? []);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Budgets</p>
          <h1>Set monthly spending limits.</h1>
        </div>
        {status && <div className="page-note">{status.month}</div>}
      </div>

      {/* Add budget */}
      <div className="panel" style={{ padding: 16 }}>
        <div className="panel-title" style={{ marginBottom: 12 }}>Add budget</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <select
            className="filter-input"
            value={newCategoryId}
            onChange={(e) => setNewCategoryId(e.target.value ? Number(e.target.value) : '')}
            style={{ flex: 1, minWidth: 160 }}
          >
            <option value="">Select category</option>
            {availableCategories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <input
            type="number"
            step="0.01"
            min="0"
            className="filter-input"
            placeholder={`Monthly limit (${currencySymbol})`}
            value={newLimit}
            onChange={(e) => setNewLimit(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            style={{ width: 160 }}
          />
          <button className="btn-primary" onClick={handleCreate}>Add</button>
        </div>
      </div>

      {/* Budget status list */}
      <div className="panel" style={{ padding: '16px' }}>
        {loading ? (
          <div className="empty">Loading...</div>
        ) : budgets.length === 0 ? (
          <div className="empty">No budgets set. Add one above.</div>
        ) : (
          <div className="list">
            {budgets.map((b) => {
              const s = statusMap.get(b.category_id);
              const spent = s?.spent ?? 0;
              const percent = s?.percent ?? 0;
              const over = percent > 100;

              return (
                <div key={b.id} className="list-row" style={{ flexWrap: 'wrap', gap: 8 }}>
                  <div style={{ flex: 1, minWidth: 140 }}>
                    <div className="list-title">{b.category_name}</div>
                    <div className="list-sub">
                      {currencySymbol}{spent.toFixed(2)} / {currencySymbol}{b.monthly_limit.toFixed(2)}
                    </div>
                  </div>

                  {/* Progress bar */}
                  <div style={{ flex: 2, minWidth: 200, display: 'flex', alignItems: 'center', gap: 8 }}>
                    <div className="budget-bar">
                      <div
                        className={`budget-fill${over ? ' budget-over' : ''}`}
                        style={{ width: `${Math.min(percent, 100)}%` }}
                      />
                    </div>
                    <span
                      className="list-sub"
                      style={{ minWidth: 42, textAlign: 'right', color: over ? 'var(--accent-rose)' : undefined, fontWeight: over ? 600 : undefined }}
                    >
                      {percent.toFixed(0)}%
                    </span>
                  </div>

                  <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    {editId === b.id ? (
                      <>
                        <input
                          type="number"
                          step="0.01"
                          className="filter-input"
                          value={editLimit}
                          onChange={(e) => setEditLimit(e.target.value)}
                          style={{ width: 100, padding: '4px 8px' }}
                        />
                        <button className="link-button" onClick={() => handleUpdate(b.id)}>Save</button>
                        <button className="link-button" onClick={() => setEditId(null)}>Cancel</button>
                      </>
                    ) : (
                      <>
                        <button className="link-button" onClick={() => { setEditId(b.id); setEditLimit(b.monthly_limit.toString()); }}>
                          Edit
                        </button>
                        <button className="link-button link-danger" onClick={() => handleDelete(b.id)}>
                          Remove
                        </button>
                      </>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
