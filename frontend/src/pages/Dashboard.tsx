import { useCallback, useEffect, useMemo, useState, type CSSProperties } from 'react';
import {
  getSummary,
  getStatsSeries,
  getMonthlySpend,
  getTransactions,
  getStatements,
  getCategories,
  getBudgetStatus,
  Summary,
  StatsSeriesResponse,
  MonthlySpendResponse,
  Transaction,
  Statement,
  Category,
  BudgetStatusResponse,
} from '../api/client';
import DateRangePicker, { type RangePreset } from '../components/DateRangePicker';
import KpiSparkline from '../components/charts/KpiSparkline';
import BalanceTrajectory from '../components/charts/BalanceTrajectory';
import CashflowRhythm, { type Bucket } from '../components/charts/CashflowRhythm';
import MonthlySpend from '../components/charts/MonthlySpend';
import ExpenseMix from '../components/charts/ExpenseMix';
import MerchantDrilldown from '../components/charts/MerchantDrilldown';
import { formatExpense, formatDate, formatCurrency } from '../utils/format';
import { useAppConfig } from '../contexts/AppConfig';

const toIsoDate = (d: Date) => d.toISOString().split('T')[0];

const bucketDaily = (daily: StatsSeriesResponse['daily']): Bucket[] => {
  if (daily.length <= 24) {
    return daily.map((item) => ({
      label: item.date,
      income: item.income,
      expenses: item.expenses,
      net: item.net,
    }));
  }

  const size = Math.ceil(daily.length / 24);
  const buckets: Bucket[] = [];

  for (let i = 0; i < daily.length; i += size) {
    const slice = daily.slice(i, i + size);
    const income = slice.reduce((sum, item) => sum + item.income, 0);
    const expenses = slice.reduce((sum, item) => sum + item.expenses, 0);
    const net = slice.reduce((sum, item) => sum + item.net, 0);
    buckets.push({ label: slice[0].date, income, expenses, net });
  }

  return buckets;
};

function presetToRange(preset: RangePreset, customStart: string, customEnd: string) {
  if (preset === 'custom') return { start: customStart, end: customEnd };

  const end = new Date();
  const start = new Date();
  switch (preset) {
    case '30d':
      start.setDate(end.getDate() - 30);
      break;
    case '90d':
      start.setDate(end.getDate() - 90);
      break;
    case '6m':
      start.setMonth(end.getMonth() - 6);
      break;
    case '1y':
      start.setFullYear(end.getFullYear() - 1);
      break;
    case 'all':
      start.setFullYear(2000);
      break;
  }
  return { start: toIsoDate(start), end: toIsoDate(end) };
}

export default function Dashboard() {
  const { currencySymbol } = useAppConfig();
  const [summary, setSummary] = useState<Summary | null>(null);
  const [series, setSeries] = useState<StatsSeriesResponse | null>(null);
  const [monthly, setMonthly] = useState<MonthlySpendResponse | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<number | 'all'>('all');
  const [recent, setRecent] = useState<Transaction[]>([]);
  const [statements, setStatements] = useState<Statement[]>([]);
  const [budgetStatus, setBudgetStatus] = useState<BudgetStatusResponse | null>(null);
  const [loading, setLoading] = useState(true);

  // Date range state
  const [rangePreset, setRangePreset] = useState<RangePreset>('90d');
  const [customStart, setCustomStart] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 90);
    return toIsoDate(d);
  });
  const [customEnd, setCustomEnd] = useState(() => toIsoDate(new Date()));

  const range = useMemo(
    () => presetToRange(rangePreset, customStart, customEnd),
    [rangePreset, customStart, customEnd],
  );

  const handleCustomChange = useCallback((start: string, end: string) => {
    setCustomStart(start);
    setCustomEnd(end);
  }, []);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const [summaryRes, seriesRes, txnRes, stmtRes, budgetRes] = await Promise.all([
          getSummary(range.start, range.end),
          getStatsSeries({ start_date: range.start, end_date: range.end }),
          getTransactions({ page: 1, page_size: 8, start_date: range.start, end_date: range.end }),
          getStatements(),
          getBudgetStatus(),
        ]);
        setSummary(summaryRes);
        setSeries(seriesRes);
        setRecent(txnRes.items);
        setStatements(stmtRes.slice(0, 5));
        setBudgetStatus(budgetRes);
      } catch (e) {
        console.error(e);
      }
      setLoading(false);
    };
    load();
  }, [range.start, range.end]);

  useEffect(() => {
    const loadMonthly = async () => {
      try {
        const [monthlyRes, categoriesRes] = await Promise.all([
          getMonthlySpend({
            start_date: range.start,
            end_date: range.end,
            category_id: selectedCategory === 'all' ? undefined : selectedCategory,
          }),
          getCategories(),
        ]);
        setMonthly(monthlyRes);
        setCategories(categoriesRes);
      } catch (e) {
        console.error(e);
      }
    };
    loadMonthly();
  }, [range.start, range.end, selectedCategory]);

  const daily = useMemo(() => series?.daily ?? [], [series?.daily]);
  const buckets = useMemo(() => bucketDaily(daily), [daily]);

  const donut = useMemo(() => {
    const cats = (series?.categories ?? [])
      .filter((item) => item.expenses > 0)
      .sort((a, b) => b.expenses - a.expenses);

    const top = cats.slice(0, 5);
    const rest = cats.slice(5);
    if (rest.length) {
      top.push({
        category_id: null,
        category_name: 'Other',
        expenses: rest.reduce((sum, item) => sum + item.expenses, 0),
        income: 0,
        net: 0,
        count: rest.reduce((sum, item) => sum + item.count, 0),
      });
    }

    const total = top.reduce((sum, item) => sum + item.expenses, 0);
    return { total, segments: top };
  }, [series?.categories]);

  const presetLabel =
    rangePreset === 'custom'
      ? `${formatDate(customStart)} - ${formatDate(customEnd)}`
      : rangePreset === 'all'
        ? 'All time'
        : `Last ${rangePreset}`;

  return (
    <div className="page">
      <div className="hero panel">
        <div className="hero-text">
          <p className="eyebrow">Expense Tracker</p>
          <h1>Know where every pound goes.</h1>
          <p className="hero-sub">
            Built for fast statement imports and grounded categorization.
          </p>
        </div>
        <div className="hero-chip">
          <div className="chip-title">Range</div>
          <div className="chip-value">{presetLabel}</div>
        </div>
      </div>

      <div className="range-sticky">
        <DateRangePicker
          preset={rangePreset}
          onPresetChange={setRangePreset}
          customStart={customStart}
          customEnd={customEnd}
          onCustomChange={handleCustomChange}
        />
      </div>

      <div className="kpi-grid">
        {[
          {
            label: 'Total Expenses',
            value: summary ? `${currencySymbol}${summary.total_expenses.toFixed(2)}` : '-',
            spark: daily.map((d) => d.expenses),
            color: '#D1495B',
          },
          {
            label: 'Daily Average',
            value: summary && daily.length > 0
              ? `${currencySymbol}${(summary.total_expenses / daily.length).toFixed(2)}`
              : '-',
            spark: daily.map((d) => d.expenses),
            color: '#3B8B80',
          },
          {
            label: 'Transactions',
            value: summary ? summary.total_transactions.toString() : '-',
            spark: daily.map((d) => d.count),
            color: '#2D6A4F',
          },
          {
            label: 'Unclassified',
            value: summary ? summary.unclassified_count.toString() : '-',
            spark: [] as number[],
            color: '#E09F3E',
          },
        ].map((item, index) => (
          <div className="panel kpi-card reveal" key={item.label} style={{ '--i': index } as CSSProperties}>
            <div className="kpi-label">{item.label}</div>
            <div className="kpi-value">{item.value}</div>
            <div className="kpi-meta">{presetLabel}</div>
            <KpiSparkline data={item.spark} color={item.color} />
          </div>
        ))}
      </div>

      <div className="grid-main">
        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Spending pace</div>
              <div className="panel-sub">This month vs last month</div>
            </div>
            <div className="panel-pill">Monthly</div>
          </div>
          <BalanceTrajectory />
        </div>

        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Daily spending</div>
              <div className="panel-sub">Spend per day with average</div>
            </div>
            <div className="panel-pill">Grouped</div>
          </div>
          <CashflowRhythm buckets={buckets} />
        </div>
      </div>

      <div className="panel chart-card monthly-chart">
        <div className="panel-header">
          <div>
            <div className="panel-title">Monthly spend</div>
            <div className="panel-sub">Track a category month by month</div>
          </div>
          <div className="panel-pill">Monthly</div>
        </div>
        <MonthlySpend
          series={monthly?.series ?? []}
          categories={categories}
          selectedCategory={selectedCategory}
          onCategoryChange={setSelectedCategory}
        />
      </div>

      <MerchantDrilldown rangeStart={range.start} rangeEnd={range.end} />

      <div className="grid-secondary">
        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Expense mix</div>
              <div className="panel-sub">Top categories by spend</div>
            </div>
            <div className="panel-pill">Top 5</div>
          </div>
          <ExpenseMix categories={donut.segments} total={donut.total} />
        </div>

        {budgetStatus && budgetStatus.items.length > 0 && (
          <div className="panel chart-card">
            <div className="panel-header">
              <div>
                <div className="panel-title">Budget tracker</div>
                <div className="panel-sub">{budgetStatus.month}</div>
              </div>
              <div className="panel-pill">{budgetStatus.items.length} budgets</div>
            </div>
            <div className="list">
              {budgetStatus.items.map((item) => {
                const over = item.percent > 100;
                return (
                  <div key={item.category_id} className="list-row" style={{ flexWrap: 'wrap', gap: 6 }}>
                    <div style={{ flex: 1, minWidth: 100 }}>
                      <div className="list-title">{item.category_name}</div>
                      <div className="list-sub">
                        {currencySymbol}{item.spent.toFixed(2)} / {currencySymbol}{item.monthly_limit.toFixed(2)}
                      </div>
                    </div>
                    <div style={{ flex: 2, minWidth: 120, display: 'flex', alignItems: 'center', gap: 6 }}>
                      <div className="budget-bar">
                        <div
                          className={`budget-fill${over ? ' budget-over' : ''}`}
                          style={{ width: `${Math.min(item.percent, 100)}%` }}
                        />
                      </div>
                      <span
                        className="list-sub"
                        style={{
                          minWidth: 36,
                          textAlign: 'right',
                          color: over ? 'var(--accent-rose)' : undefined,
                          fontWeight: over ? 600 : undefined,
                        }}
                      >
                        {item.percent.toFixed(0)}%
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Recent transactions</div>
              <div className="panel-sub">Latest imported activity</div>
            </div>
            <div className="panel-pill">{loading ? 'Loading' : `${recent.length} items`}</div>
          </div>
          <div className="list">
            {recent.map((txn) => (
              <div key={txn.id} className="list-row">
                <div>
                  <div className="list-title">{txn.raw_description}</div>
                  <div className="list-sub">{formatDate(txn.raw_date)}</div>
                </div>
                <div className={`list-amount${txn.amount > 0 ? ' pos' : ''}`}>
                  {formatExpense(txn.amount, currencySymbol)}
                </div>
              </div>
            ))}
            {recent.length === 0 && <div className="empty">No transactions yet.</div>}
          </div>
        </div>

        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Recent statements</div>
              <div className="panel-sub">Import history at a glance</div>
            </div>
            <div className="panel-pill">{statements.length} files</div>
          </div>
          <div className="list">
            {statements.map((stmt) => (
              <div key={stmt.id} className="list-row">
                <div>
                  <div className="list-title">{stmt.filename}</div>
                  <div className="list-sub">
                    {formatDate(stmt.period_start)} - {formatDate(stmt.period_end)}
                  </div>
                </div>
                <div className="list-meta">
                  {stmt.closing_balance !== null ? formatCurrency(stmt.closing_balance, currencySymbol) : '-'}
                </div>
              </div>
            ))}
            {statements.length === 0 && <div className="empty">No statements imported.</div>}
          </div>
        </div>
      </div>
    </div>
  );
}
