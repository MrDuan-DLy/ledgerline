import { useState, useEffect } from 'react';
import { getCategories, createCategory, deleteCategory, Category } from '../api/client';
import { toast } from '../components/Toast';

interface TreeNode extends Category {
  children: TreeNode[];
}

function buildTree(cats: Category[]): TreeNode[] {
  const map = new Map<number, TreeNode>();
  cats.forEach((c) => map.set(c.id, { ...c, children: [] }));

  const roots: TreeNode[] = [];
  map.forEach((node) => {
    if (node.parent_id && map.has(node.parent_id)) {
      map.get(node.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  });

  roots.sort((a, b) => a.name.localeCompare(b.name));
  roots.forEach((r) => r.children.sort((a, b) => a.name.localeCompare(b.name)));
  return roots;
}

export default function Categories() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(true);
  const [newName, setNewName] = useState('');
  const [newParent, setNewParent] = useState<number | ''>('');
  const [newIsExpense, setNewIsExpense] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      setCategories(await getCategories());
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Failed to load', 'error');
    }
    setLoading(false);
  };

  useEffect(() => { load(); }, []);

  const tree = buildTree(categories);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    try {
      await createCategory({
        name: newName.trim(),
        parent_id: newParent || null,
        is_expense: newIsExpense,
      });
      setNewName('');
      setNewParent('');
      load();
      toast('Category created', 'success');
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Failed to create', 'error');
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Delete "${name}"?`)) return;
    try {
      await deleteCategory(id);
      load();
      toast(`Deleted "${name}"`, 'success');
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Failed to delete', 'error');
    }
  };

  const renderNode = (node: TreeNode, depth = 0) => (
    <div key={node.id}>
      <div
        className="list-row"
        style={{ paddingLeft: depth * 24 + 12 }}
      >
        <div style={{ flex: 1 }}>
          <div className="list-title">{node.name}</div>
          <div className="list-sub">
            {node.is_expense ? 'Expense' : 'Income'}
            {node.parent_id ? '' : ' (top-level)'}
          </div>
        </div>
        <button
          className="link-button link-danger"
          onClick={() => handleDelete(node.id, node.name)}
        >
          Delete
        </button>
      </div>
      {node.children.map((child) => renderNode(child, depth + 1))}
    </div>
  );

  // Top-level categories for parent dropdown
  const parentOptions = categories.filter((c) => c.parent_id === null);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Categories</p>
          <h1>Organize your spending.</h1>
        </div>
      </div>

      <div className="panel" style={{ padding: 16 }}>
        <div className="panel-title" style={{ marginBottom: 12 }}>Add category</div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <input
            type="text"
            className="filter-input"
            placeholder="Category name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            style={{ flex: 1, minWidth: 160 }}
          />
          <select
            className="filter-input"
            value={newParent}
            onChange={(e) => setNewParent(e.target.value ? Number(e.target.value) : '')}
            style={{ flex: 0, minWidth: 140 }}
          >
            <option value="">No parent (top-level)</option>
            {parentOptions.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </select>
          <label className="filter-check">
            <input
              type="checkbox"
              checked={newIsExpense}
              onChange={(e) => setNewIsExpense(e.target.checked)}
            />
            Expense
          </label>
          <button className="btn-primary" onClick={handleCreate}>
            Add
          </button>
        </div>
      </div>

      <div className="panel" style={{ padding: '12px 0' }}>
        {loading ? (
          <div className="empty" style={{ padding: 20 }}>Loading...</div>
        ) : tree.length === 0 ? (
          <div className="empty" style={{ padding: 20 }}>No categories.</div>
        ) : (
          <div className="list">
            {tree.map((node) => renderNode(node))}
          </div>
        )}
      </div>
    </div>
  );
}
