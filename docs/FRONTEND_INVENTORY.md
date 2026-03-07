# Frontend Inventory

**Brand**: Ledgerline
**Framework**: React 18 + Vite + TailwindCSS v4
**Charts**: Recharts

## Pages

### Dashboard (`/`)
**File**: `src/pages/Dashboard.tsx`

- Date range picker with presets (30d, 90d, 6m, 1y, all, custom)
- KPI cards: Total Expenses, Daily Average, Transactions, Unclassified
- Spending pace chart (current vs previous month)
- Daily cashflow rhythm chart with average reference line
- Monthly spend chart with category filter and trend line
- Merchant drilldown component
- Expense mix donut chart (top 5 categories + "Other")
- Budget tracker with progress bars
- Recent transactions list (8)
- Recent statements list (5)

**API calls**: `getSummary`, `getStatsSeries`, `getMonthlySpend`, `getTransactions`, `getStatements`, `getCategories`, `getBudgetStatus`

### Transactions (`/transactions`)
**File**: `src/pages/Transactions.tsx`

- Paginated table (50/page) with search
- Filters: unclassified only, hide excluded, statement_id (URL param)
- Bulk selection: delete, exclude/include, classify
- Inline category assignment dropdown
- Edit transaction details (date, description)
- Expandable row showing linked receipt + receipt items
- Summary KPI cards

**API calls**: `getTransactions`, `getCategories`, `getSummary`, `updateTransaction`, `deleteTransaction`, `bulkDelete`, `bulkExclude`, `bulkClassify`, `getReceiptByTransaction`, `updateReceiptItem`

### Import (`/import`)
**File**: `src/pages/Import.tsx`

- Drag & drop file upload (PDF/CSV)
- PDF -> AI review flow routing
- CSV -> direct import
- Pending review sessions table
- Import history (confirmed statements)
- AI import history (reviewed PDFs)
- Session deletion

**API calls**: `uploadStatement`, `uploadForReview`, `getStatements`, `getImportSessions`, `deleteImportSession`

### Review (`/review/:sessionId`)
**File**: `src/pages/Review.tsx`

- Split layout: source preview (left) + extracted items (right)
- PDF page navigation with image preview
- AI usage stats (tokens, duration, model)
- PDF metadata KPIs (period, balances, count)
- Item editing (date, description, amount)
- Duplicate detection display
- Skip/confirm per item
- Receipt-specific detail card and line items
- Summary footer (total, imported, skipped, duplicates)
- Read-only mode for confirmed sessions

**API calls**: `getImportSession`, `getCategories`, `updateImportItem`, `confirmImportSession`, `deleteImportSession`, `importSourceUrl`, `importPageUrl`

### Receipts (`/receipts`)
**File**: `src/pages/Receipts.tsx`

- Batch upload (JPG, PNG, WebP)
- Receipt image preview
- OCR result editing (merchant, date, amount, currency, payment method)
- Line items table (editable name/total)
- Suggested transaction matching with link action
- Pagination (10/page)
- Status indicators (pending/confirmed)

**API calls**: `uploadReceipts`, `getReceipts`, `getCategories`, `confirmReceipt`, `receiptImageUrl`

### Categories (`/categories`)
**File**: `src/pages/Categories.tsx`

- Category tree display (hierarchical with indent)
- Add form: name, parent, expense/income toggle
- Delete with confirmation
- Alphabetical sorting

**API calls**: `getCategories`, `createCategory`, `deleteCategory`

### Budgets (`/budgets`)
**File**: `src/pages/Budgets.tsx`

- Create form: category selector + monthly limit
- Budget status list with progress bars
- Inline edit (limit amount)
- Delete with confirmation
- Over-budget color coding

**API calls**: `getBudgets`, `getBudgetStatus`, `getCategories`, `createBudget`, `updateBudget`, `deleteBudget`

## Components

### General

| Component | File | Props | Used In |
|-----------|------|-------|---------|
| Toast | `components/Toast.tsx` | none (provider) | App.tsx (root) |
| DateRangePicker | `components/DateRangePicker.tsx` | `preset`, `onPresetChange`, `customStart`, `customEnd`, `onCustomChange` | Dashboard |
| ChartTooltip | `components/ChartTooltip.tsx` | `active`, `payload`, `label` | All chart components |

### Charts (`components/charts/`)

| Component | File | Props | Description |
|-----------|------|-------|-------------|
| KpiSparkline | `charts/KpiSparkline.tsx` | `data: number[]`, `color: string` | Mini area chart (36px) for KPI cards |
| BalanceTrajectory | `charts/BalanceTrajectory.tsx` | none (self-contained) | Current vs previous month spending pace |
| CashflowRhythm | `charts/CashflowRhythm.tsx` | `buckets: Bucket[]` | Daily expense bars with average line |
| MonthlySpend | `charts/MonthlySpend.tsx` | `series`, `categories`, `selectedCategory`, `onCategoryChange` | Monthly bars with 2-month moving average |
| ExpenseMix | `charts/ExpenseMix.tsx` | `categories: CategoryTotal[]`, `total: number` | Donut chart with legend |
| MerchantDrilldown | `charts/MerchantDrilldown.tsx` | `rangeStart`, `rangeEnd` | Merchant selector + spending bar chart |

## API Client

**File**: `src/api/client.ts`
**Base URL**: `/api`

40+ typed methods covering: Transactions, Statistics, Categories, Budgets, Merchants, Statements, Receipts, Import Sessions, Rules.

Key interfaces exported: `Transaction`, `Category`, `Statement`, `Receipt`, `ImportSession`, `BudgetResponse`, `BudgetStatusResponse`, `MerchantResponse`.

## Utilities

**File**: `src/utils/format.ts`

| Function | Signature | Description |
|----------|-----------|-------------|
| `formatExpense` | `(amount: number) => string` | Negative -> `£123.45`, positive -> `+£10.00` |
| `formatDate` | `(dateStr: string \| null) => string` | Format to "02 Mar 2026" (en-GB) |
| `formatCurrency` | `(value: number) => string` | Format with sign preserved |

## Routing (App.tsx)

| Path | Page | Notes |
|------|------|-------|
| `/` | Dashboard | Default |
| `/transactions` | Transactions | Supports `?statement_id=` |
| `/import` | Import | |
| `/review/:sessionId` | Review | AI import review |
| `/receipts` | Receipts | |
| `/categories` | Categories | |
| `/budgets` | Budgets | |

## Navigation

Persistent sidebar with NavLink (active state):
Dashboard, Transactions, Import, Receipts, Categories, Budgets

## Patterns

- **State**: React hooks (no global state library)
- **Data fetching**: async/await with try/catch + toast errors
- **Styling**: TailwindCSS classes + index.css custom styles
- **Forms**: Controlled inputs with draft state
- **Bulk ops**: Set-based selection
- **Charts**: Recharts with ResponsiveContainer
