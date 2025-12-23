const API_BASE = 'http://localhost:8000/api';

export interface Transaction {
  id: number;
  statement_id: number | null;
  source_hash: string;
  raw_date: string;
  raw_description: string;
  raw_amount: number;
  raw_balance: number | null;
  effective_date: string | null;
  description: string | null;
  amount: number;
  category_id: number | null;
  category_name: string | null;
  category_source: string;
  is_reconciled: boolean;
  reconciled_at: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface TransactionListResponse {
  items: Transaction[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface Category {
  id: number;
  name: string;
  parent_id: number | null;
  is_expense: boolean;
}

export interface Statement {
  id: number;
  account_id: string;
  filename: string;
  period_start: string;
  period_end: string;
  opening_balance: number | null;
  closing_balance: number | null;
  imported_at: string;
  transaction_count: number;
}

export interface ImportResult {
  success: boolean;
  statement_id: number | null;
  transactions_imported: number;
  transactions_skipped: number;
  errors: string[];
  message: string;
}

export interface ReceiptItem {
  id: number;
  name: string;
  quantity: number | null;
  unit_price: number | null;
  line_total: number | null;
}

export interface Receipt {
  id: number;
  image_path: string;
  image_hash: string;
  merchant_name: string | null;
  receipt_date: string | null;
  receipt_time: string | null;
  total_amount: number | null;
  currency: string | null;
  payment_method: string | null;
  status: string;
  ocr_raw: string | null;
  ocr_json: string | null;
  transaction_id: number | null;
  matched_transaction_id: number | null;
  matched_transaction_date: string | null;
  matched_transaction_amount: number | null;
  matched_transaction_description: string | null;
  matched_reason: string | null;
  created_at: string;
  items: ReceiptItem[];
}

export interface ReceiptUploadResult {
  success: boolean;
  receipt_id: number | null;
  message: string;
  errors: string[];
}

export interface Summary {
  total_transactions: number;
  total_income: number;
  total_expenses: number;
  net: number;
  unclassified_count: number;
}

export interface DailySeriesPoint {
  date: string;
  net: number;
  income: number;
  expenses: number;
  count: number;
  cumulative: number;
}

export interface CategoryTotal {
  category_id: number | null;
  category_name: string;
  expenses: number;
  income: number;
  net: number;
  count: number;
}

export interface StatsSeriesResponse {
  start_date: string | null;
  end_date: string | null;
  daily: DailySeriesPoint[];
  categories: CategoryTotal[];
}

// Transactions
export async function getTransactions(params: {
  page?: number;
  page_size?: number;
  start_date?: string;
  end_date?: string;
  category_id?: number;
  unclassified_only?: boolean;
  search?: string;
}): Promise<TransactionListResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', params.page.toString());
  if (params.page_size) query.set('page_size', params.page_size.toString());
  if (params.start_date) query.set('start_date', params.start_date);
  if (params.end_date) query.set('end_date', params.end_date);
  if (params.category_id) query.set('category_id', params.category_id.toString());
  if (params.unclassified_only) query.set('unclassified_only', 'true');
  if (params.search) query.set('search', params.search);

  const res = await fetch(`${API_BASE}/transactions?${query}`);
  return res.json();
}

export async function updateTransaction(id: number, data: {
  category_id?: number;
  notes?: string;
  effective_date?: string;
}): Promise<Transaction> {
  const res = await fetch(`${API_BASE}/transactions/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function bulkClassify(ids: number[], category_id: number): Promise<{ updated: number }> {
  const res = await fetch(`${API_BASE}/transactions/bulk-classify?category_id=${category_id}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ids),
  });
  return res.json();
}

export async function getSummary(start_date?: string, end_date?: string): Promise<Summary> {
  const query = new URLSearchParams();
  if (start_date) query.set('start_date', start_date);
  if (end_date) query.set('end_date', end_date);
  const res = await fetch(`${API_BASE}/transactions/stats/summary?${query}`);
  return res.json();
}

export async function getStatsSeries(params: {
  start_date?: string;
  end_date?: string;
}): Promise<StatsSeriesResponse> {
  const query = new URLSearchParams();
  if (params.start_date) query.set('start_date', params.start_date);
  if (params.end_date) query.set('end_date', params.end_date);
  const res = await fetch(`${API_BASE}/transactions/stats/series?${query}`);
  return res.json();
}

// Categories
export async function getCategories(): Promise<Category[]> {
  const res = await fetch(`${API_BASE}/categories`);
  return res.json();
}

// Statements
export async function getStatements(): Promise<Statement[]> {
  const res = await fetch(`${API_BASE}/statements`);
  return res.json();
}

export async function uploadStatement(file: File): Promise<ImportResult> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${API_BASE}/statements/upload`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

// Receipts
export async function uploadReceipt(file: File): Promise<ReceiptUploadResult> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await fetch(`${API_BASE}/receipts/upload`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

export async function uploadReceipts(files: File[]): Promise<ReceiptUploadResult[]> {
  const formData = new FormData();
  files.forEach((file) => formData.append('files', file));

  const res = await fetch(`${API_BASE}/receipts/upload-batch`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

export async function getReceipts(): Promise<Receipt[]> {
  const res = await fetch(`${API_BASE}/receipts`);
  return res.json();
}

export async function confirmReceipt(id: number, data: {
  merchant_name?: string;
  receipt_date?: string;
  total_amount?: number;
  currency?: string;
  category_id?: number;
  notes?: string;
  transaction_id?: number;
}): Promise<{ transaction_id: number; receipt_id: number }> {
  const res = await fetch(`${API_BASE}/receipts/${id}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

// Rules
export async function reclassifyAll(): Promise<{ updated: number }> {
  const res = await fetch(`${API_BASE}/rules/reclassify`, { method: 'POST' });
  return res.json();
}
