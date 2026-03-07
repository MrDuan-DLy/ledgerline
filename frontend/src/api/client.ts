const API_BASE = '/api';

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}

async function apiFetch(url: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(url, init);
  if (!res.ok) {
    let message = `Request failed: ${res.status}`;
    try {
      const body = await res.json();
      if (body.detail) message = body.detail;
      else if (body.message) message = body.message;
    } catch {
      // use default message
    }
    throw new ApiError(res.status, message);
  }
  return res;
}

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
  is_excluded: boolean;
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

export interface MonthlySpendPoint {
  month: string;
  total_expenses: number;
}

export interface MonthlySpendResponse {
  start_date: string | null;
  end_date: string | null;
  category_id: number | null;
  series: MonthlySpendPoint[];
}

// Budgets
export interface BudgetResponse {
  id: number;
  category_id: number;
  category_name: string | null;
  monthly_limit: number;
  created_at: string;
}

export interface BudgetStatusItem {
  category_id: number;
  category_name: string;
  monthly_limit: number;
  spent: number;
  remaining: number;
  percent: number;
}

export interface BudgetStatusResponse {
  month: string;
  items: BudgetStatusItem[];
}

// App config
export interface AppConfigResponse {
  currency_symbol: string;
  locale: string;
  supported_formats: string[];
  app_name: string;
}

export async function getConfig(): Promise<AppConfigResponse> {
  const res = await fetch(`${API_BASE}/config`);
  if (!res.ok) throw new Error('Config unavailable');
  return res.json();
}

export async function getBudgets(): Promise<BudgetResponse[]> {
  const res = await apiFetch(`${API_BASE}/budgets`);
  return res.json();
}

export async function createBudget(data: { category_id: number; monthly_limit: number }): Promise<BudgetResponse> {
  const res = await apiFetch(`${API_BASE}/budgets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateBudget(id: number, data: { monthly_limit: number }): Promise<BudgetResponse> {
  const res = await apiFetch(`${API_BASE}/budgets/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteBudget(id: number): Promise<{ deleted: boolean }> {
  const res = await apiFetch(`${API_BASE}/budgets/${id}`, { method: 'DELETE' });
  return res.json();
}

export async function getBudgetStatus(): Promise<BudgetStatusResponse> {
  const res = await apiFetch(`${API_BASE}/budgets/status`);
  return res.json();
}

// Spending Pace
export interface PaceDayPoint {
  day: number;
  cumulative: number;
}

export interface SpendingPaceResponse {
  current_month: string;
  current_series: PaceDayPoint[];
  prev_month: string;
  prev_series: PaceDayPoint[];
}

export async function getSpendingPace(): Promise<SpendingPaceResponse> {
  const res = await apiFetch(`${API_BASE}/transactions/stats/pace`);
  return res.json();
}

// Merchants
export interface Merchant {
  id: number;
  name: string;
  patterns: string[];
  category_id: number | null;
  transaction_count: number | null;
  created_at: string;
}

export interface MerchantMatchResult {
  canonical_name: string | null;
  merchant_id: number | null;
  score: number;
  match_type: string;
}

export interface MerchantTransaction {
  id: number;
  date: string;
  description: string;
  amount: number;
  category_name: string | null;
}

export interface MerchantTransactionsResponse {
  merchant: Merchant;
  total_spend: number;
  count: number;
  transactions: MerchantTransaction[];
}

export async function getMerchants(withCounts?: boolean): Promise<Merchant[]> {
  const query = withCounts ? '?with_counts=true' : '';
  const res = await apiFetch(`${API_BASE}/merchants/${query}`);
  return res.json();
}

export async function getMerchantTransactions(
  merchantId: number,
  params?: { start_date?: string; end_date?: string },
): Promise<MerchantTransactionsResponse> {
  const query = new URLSearchParams();
  if (params?.start_date) query.set('start_date', params.start_date);
  if (params?.end_date) query.set('end_date', params.end_date);
  const res = await apiFetch(`${API_BASE}/merchants/${merchantId}/transactions?${query}`);
  return res.json();
}

export interface BackfillResult {
  success: boolean;
  transactions_updated: number;
  patterns_learned: number;
  total_scanned: number;
  message: string;
}

export async function backfillMerchantNames(): Promise<BackfillResult> {
  const res = await apiFetch(`${API_BASE}/merchants/backfill`, { method: 'POST' });
  return res.json();
}

export async function matchMerchant(rawName: string): Promise<MerchantMatchResult> {
  const res = await apiFetch(`${API_BASE}/merchants/match?raw_name=${encodeURIComponent(rawName)}`);
  return res.json();
}

export async function createMerchant(data: {
  name: string;
  patterns?: string[];
  category_id?: number;
}): Promise<Merchant> {
  const res = await apiFetch(`${API_BASE}/merchants/`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateMerchant(id: number, data: {
  name?: string;
  patterns?: string[];
  category_id?: number;
}): Promise<Merchant> {
  const res = await apiFetch(`${API_BASE}/merchants/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteMerchant(id: number): Promise<{ success: boolean; message: string }> {
  const res = await apiFetch(`${API_BASE}/merchants/${id}`, {
    method: 'DELETE',
  });
  return res.json();
}

export async function mergeMerchant(targetId: number, sourceId: number): Promise<Merchant> {
  const res = await apiFetch(`${API_BASE}/merchants/${targetId}/merge`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ source_merchant_id: sourceId }),
  });
  return res.json();
}

// Transactions
export async function getTransactions(params: {
  page?: number;
  page_size?: number;
  start_date?: string;
  end_date?: string;
  category_id?: number;
  unclassified_only?: boolean;
  excluded_only?: boolean;
  hide_excluded?: boolean;
  search?: string;
  statement_id?: number;
}): Promise<TransactionListResponse> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', params.page.toString());
  if (params.page_size) query.set('page_size', params.page_size.toString());
  if (params.start_date) query.set('start_date', params.start_date);
  if (params.end_date) query.set('end_date', params.end_date);
  if (params.category_id) query.set('category_id', params.category_id.toString());
  if (params.unclassified_only) query.set('unclassified_only', 'true');
  if (params.excluded_only) query.set('excluded_only', 'true');
  if (params.hide_excluded) query.set('hide_excluded', 'true');
  if (params.search) query.set('search', params.search);
  if (params.statement_id) query.set('statement_id', params.statement_id.toString());

  const res = await apiFetch(`${API_BASE}/transactions?${query}`);
  return res.json();
}

export async function bulkExclude(ids: number[], exclude: boolean = true): Promise<{ updated: number }> {
  const res = await apiFetch(`${API_BASE}/transactions/bulk-exclude?exclude=${exclude}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ids),
  });
  return res.json();
}

export async function updateTransaction(id: number, data: {
  category_id?: number;
  notes?: string;
  effective_date?: string;
  description?: string;
  is_excluded?: boolean;
}): Promise<Transaction> {
  const res = await apiFetch(`${API_BASE}/transactions/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteTransaction(id: number): Promise<{ success: boolean; message: string }> {
  const res = await apiFetch(`${API_BASE}/transactions/${id}`, { method: 'DELETE' });
  return res.json();
}

export async function bulkDelete(ids: number[]): Promise<{ deleted: number }> {
  const res = await apiFetch(`${API_BASE}/transactions/bulk-delete`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(ids),
  });
  return res.json();
}

export async function bulkClassify(ids: number[], category_id: number): Promise<{ updated: number }> {
  const res = await apiFetch(`${API_BASE}/transactions/bulk-classify?category_id=${category_id}`, {
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
  const res = await apiFetch(`${API_BASE}/transactions/stats/summary?${query}`);
  return res.json();
}

export async function getStatsSeries(params: {
  start_date?: string;
  end_date?: string;
}): Promise<StatsSeriesResponse> {
  const query = new URLSearchParams();
  if (params.start_date) query.set('start_date', params.start_date);
  if (params.end_date) query.set('end_date', params.end_date);
  const res = await apiFetch(`${API_BASE}/transactions/stats/series?${query}`);
  return res.json();
}

export async function getMonthlySpend(params: {
  start_date?: string;
  end_date?: string;
  category_id?: number;
}): Promise<MonthlySpendResponse> {
  const query = new URLSearchParams();
  if (params.start_date) query.set('start_date', params.start_date);
  if (params.end_date) query.set('end_date', params.end_date);
  if (params.category_id) query.set('category_id', params.category_id.toString());
  const res = await apiFetch(`${API_BASE}/transactions/stats/monthly?${query}`);
  return res.json();
}

// Categories
export async function getCategories(): Promise<Category[]> {
  const res = await apiFetch(`${API_BASE}/categories`);
  return res.json();
}

export async function createCategory(data: {
  name: string;
  parent_id?: number | null;
  is_expense?: boolean;
}): Promise<Category> {
  const res = await apiFetch(`${API_BASE}/categories`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function deleteCategory(id: number): Promise<{ deleted: boolean }> {
  const res = await apiFetch(`${API_BASE}/categories/${id}`, { method: 'DELETE' });
  return res.json();
}

// Statements
export async function getStatements(): Promise<Statement[]> {
  const res = await apiFetch(`${API_BASE}/statements`);
  return res.json();
}

export async function uploadStatement(file: File): Promise<ImportResult> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await apiFetch(`${API_BASE}/statements/upload`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

// Receipts
export async function uploadReceipt(file: File): Promise<ReceiptUploadResult> {
  const formData = new FormData();
  formData.append('file', file);

  const res = await apiFetch(`${API_BASE}/receipts/upload`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

export async function uploadReceipts(files: File[]): Promise<ReceiptUploadResult[]> {
  const formData = new FormData();
  files.forEach((file) => formData.append('files', file));

  const res = await apiFetch(`${API_BASE}/receipts/upload-batch`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

export async function getReceipts(): Promise<Receipt[]> {
  const res = await apiFetch(`${API_BASE}/receipts`);
  return res.json();
}

export async function getReceiptByTransaction(transactionId: number): Promise<Receipt | null> {
  const res = await fetch(`${API_BASE}/receipts/by-transaction/${transactionId}`);
  if (!res.ok) return null;
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
  const res = await apiFetch(`${API_BASE}/receipts/${id}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function updateReceiptItem(id: number, data: {
  name?: string;
  quantity?: number | null;
  unit_price?: number | null;
  line_total?: number | null;
}): Promise<{ updated: boolean }> {
  const res = await apiFetch(`${API_BASE}/receipts/items/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export function receiptImageUrl(receiptId: number): string {
  return `${API_BASE}/receipts/${receiptId}/image`;
}

// Rules
export async function reclassifyAll(): Promise<{ updated: number }> {
  const res = await apiFetch(`${API_BASE}/rules/reclassify`, { method: 'POST' });
  return res.json();
}

// Import Sessions
export interface ImportItem {
  id: number;
  session_id: string;
  page_num: number | null;
  extracted_date: string | null;
  extracted_description: string | null;
  extracted_amount: number | null;
  extracted_balance: number | null;
  extracted_merchant: string | null;
  extracted_items_json: string | null;
  status: string;
  duplicate_of_id: number | null;
  duplicate_score: number | null;
  duplicate_reason: string | null;
  duplicate_transaction_date: string | null;
  duplicate_transaction_description: string | null;
  duplicate_transaction_amount: number | null;
  created_at: string;
}

export interface ImportSession {
  id: string;
  source_type: string;
  source_file: string;
  file_hash: string;
  page_count: number | null;
  page_image_paths: string[] | null;
  metadata_json: string | null;
  ai_usage_json: string | null;
  status: string;
  created_at: string;
  items: ImportItem[];
}

export interface ImportUploadResult {
  success: boolean;
  session_id: string | null;
  message: string;
  errors: string[];
}

export interface ConfirmResult {
  success: boolean;
  created: number;
  linked: number;
  skipped: number;
  message: string;
}

export async function getImportSessions(): Promise<ImportSession[]> {
  const res = await apiFetch(`${API_BASE}/imports/`);
  return res.json();
}

export async function deleteImportSession(sessionId: string): Promise<{ success: boolean; message: string }> {
  const res = await apiFetch(`${API_BASE}/imports/${sessionId}`, {
    method: 'DELETE',
  });
  return res.json();
}

export function importSourceUrl(sessionId: string): string {
  return `${API_BASE}/imports/${sessionId}/source`;
}

export function importPageUrl(sessionId: string, pageNum: number): string {
  return `${API_BASE}/imports/${sessionId}/pages/${pageNum}`;
}

export async function uploadForReview(file: File): Promise<ImportUploadResult> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await apiFetch(`${API_BASE}/imports/upload`, {
    method: 'POST',
    body: formData,
  });
  return res.json();
}

export async function getImportSession(sessionId: string): Promise<ImportSession> {
  const res = await apiFetch(`${API_BASE}/imports/${sessionId}`);
  return res.json();
}

export async function updateImportItem(
  sessionId: string,
  itemId: number,
  data: {
    extracted_date?: string;
    extracted_description?: string;
    extracted_amount?: number;
    extracted_balance?: number;
    extracted_merchant?: string;
    status?: string;
    duplicate_of_id?: number | null;
  }
): Promise<ImportItem> {
  const res = await apiFetch(`${API_BASE}/imports/${sessionId}/items/${itemId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  return res.json();
}

export async function confirmImportSession(sessionId: string): Promise<ConfirmResult> {
  const res = await apiFetch(`${API_BASE}/imports/${sessionId}/confirm`, {
    method: 'POST',
  });
  return res.json();
}

export async function discardImportSession(sessionId: string): Promise<{ success: boolean; message: string }> {
  const res = await apiFetch(`${API_BASE}/imports/${sessionId}`, {
    method: 'DELETE',
  });
  return res.json();
}
