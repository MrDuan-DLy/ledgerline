import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import {
  getSummary,
  getStatsSeries,
  getTransactions,
  getStatements,
  Summary,
  StatsSeriesResponse,
  Transaction,
  Statement,
} from '../api/client';

const CHART_COLORS = ['#2D6A4F', '#3B8B80', '#E09F3E', '#D1495B', '#6D4C41', '#7A6C5D'];

const toIsoDate = (d: Date) => d.toISOString().split('T')[0];

const formatCurrency = (value: number) => {
  const sign = value < 0 ? '-' : '';
  return `${sign}£${Math.abs(value).toFixed(2)}`;
};

const formatDate = (dateStr: string) =>
  new Date(dateStr).toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });

type Bucket = {
  label: string;
  income: number;
  expenses: number;
  net: number;
};

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
    buckets.push({
      label: slice[0].date,
      income,
      expenses,
      net,
    });
  }

  return buckets;
};

export default function Dashboard() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [series, setSeries] = useState<StatsSeriesResponse | null>(null);
  const [recent, setRecent] = useState<Transaction[]>([]);
  const [statements, setStatements] = useState<Statement[]>([]);
  const [loading, setLoading] = useState(true);
  const [lineHoverIndex, setLineHoverIndex] = useState<number | null>(null);
  const [barHoverIndex, setBarHoverIndex] = useState<number | null>(null);

  const range = useMemo(() => {
    const end = new Date();
    const start = new Date();
    start.setDate(end.getDate() - 90);
    return { start: toIsoDate(start), end: toIsoDate(end) };
  }, []);

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        const [summaryRes, seriesRes, txnRes, stmtRes] = await Promise.all([
          getSummary(range.start, range.end),
          getStatsSeries({ start_date: range.start, end_date: range.end }),
          getTransactions({ page: 1, page_size: 8 }),
          getStatements(),
        ]);
        setSummary(summaryRes);
        setSeries(seriesRes);
        setRecent(txnRes.items);
        setStatements(stmtRes.slice(0, 5));
      } catch (e) {
        console.error(e);
      }
      setLoading(false);
    };
    load();
  }, [range.end, range.start]);

  const daily = series?.daily ?? [];
  const buckets = useMemo(() => bucketDaily(daily), [daily]);

  const lineData = useMemo(() => {
    if (daily.length < 2) return { path: '', points: [] as { x: number; y: number }[] };
    const values = daily.map((item) => item.cumulative);
    const min = Math.min(...values);
    const max = Math.max(...values);
    const width = 600;
    const height = 220;
    const pad = 24;
    const rangeVal = max - min || 1;
    const points = values.map((value, index) => {
      const x = pad + (index / (values.length - 1)) * (width - pad * 2);
      const y = height - pad - ((value - min) / rangeVal) * (height - pad * 2);
      return { x, y };
    });

    const path = points
      .map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`)
      .join(' ');
    return { path, points };
  }, [daily]);

  const areaPath = useMemo(() => {
    if (!lineData.path) return '';
    const width = 600;
    const height = 220;
    const pad = 24;
    return `${lineData.path} L ${width - pad} ${height - pad} L ${pad} ${height - pad} Z`;
  }, [lineData.path]);

  const monthTicks = useMemo(() => {
    if (!daily.length) return [];
    const ticks: { label: string; index: number }[] = [];
    let lastMonth = '';
    daily.forEach((item, index) => {
      const d = new Date(item.date);
      const label = d.toLocaleDateString('en-GB', { month: 'short' });
      const monthKey = `${d.getFullYear()}-${d.getMonth()}`;
      if (monthKey !== lastMonth) {
        ticks.push({ label, index });
        lastMonth = monthKey;
      }
    });
    return ticks;
  }, [daily]);

  const bucketMonthTicks = useMemo(() => {
    if (!buckets.length) return [];
    const ticks: { label: string; index: number }[] = [];
    let lastMonth = '';
    buckets.forEach((bucket, index) => {
      const d = new Date(bucket.label);
      const label = d.toLocaleDateString('en-GB', { month: 'short' });
      const monthKey = `${d.getFullYear()}-${d.getMonth()}`;
      if (monthKey !== lastMonth) {
        ticks.push({ label, index });
        lastMonth = monthKey;
      }
    });
    return ticks;
  }, [buckets]);

  const donut = useMemo(() => {
    const categories = (series?.categories ?? [])
      .filter((item) => item.expenses > 0)
      .sort((a, b) => b.expenses - a.expenses);

    const top = categories.slice(0, 5);
    const rest = categories.slice(5);
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

  return (
    <div className="page">
      <div className="hero panel">
        <div className="hero-text">
          <p className="eyebrow">Personal Finance</p>
          <h1>Clarity over cashflow, without the clutter.</h1>
          <p className="hero-sub">
            Built for fast statement imports and grounded categorization. Showing the last 90 days.
          </p>
        </div>
        <div className="hero-chip">
          <div className="chip-title">Range</div>
          <div className="chip-value">{formatDate(range.start)} - {formatDate(range.end)}</div>
          <div className="chip-note">Auto-refresh on import</div>
        </div>
      </div>

      <div className="kpi-grid">
        {[
          { label: 'Total Income', value: summary ? `£${summary.total_income.toFixed(2)}` : '-' },
          { label: 'Total Expenses', value: summary ? `£${summary.total_expenses.toFixed(2)}` : '-' },
          { label: 'Net Movement', value: summary ? formatCurrency(summary.net) : '-' },
          { label: 'Unclassified', value: summary ? summary.unclassified_count.toString() : '-' },
        ].map((item, index) => (
          <div className="panel kpi-card reveal" key={item.label} style={{ '--i': index } as CSSProperties}>
            <div className="kpi-label">{item.label}</div>
            <div className="kpi-value">{item.value}</div>
            <div className="kpi-meta">Last 90 days</div>
          </div>
        ))}
      </div>

      <div className="grid-main">
        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Balance trajectory</div>
              <div className="panel-sub">Cumulative net movement by day</div>
            </div>
            <div className="panel-pill">
              {lineHoverIndex !== null && daily[lineHoverIndex]
                ? `${formatDate(daily[lineHoverIndex].date)} · ${formatCurrency(daily[lineHoverIndex].cumulative)}`
                : `${daily.length} days`}
            </div>
          </div>
          {daily.length === 0 ? (
            <div className="empty">No data yet. Import a statement to see movement.</div>
          ) : (
            <svg
              viewBox="0 0 600 220"
              className="chart-svg chart-hover"
              onMouseLeave={() => setLineHoverIndex(null)}
              onMouseMove={(event) => {
                const rect = (event.currentTarget as SVGSVGElement).getBoundingClientRect();
                const width = 600;
                const pad = 24;
                const ratio = (event.clientX - rect.left) / rect.width;
                const x = ratio * width;
                const index = Math.round(((x - pad) / (width - pad * 2)) * (daily.length - 1));
                if (index >= 0 && index < daily.length) {
                  setLineHoverIndex(index);
                }
              }}
            >
              <defs>
                <linearGradient id="areaFade" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="0%" stopColor="#3B8B80" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#3B8B80" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path d={areaPath} fill="url(#areaFade)" />
              <path d={lineData.path} fill="none" stroke="#1B4332" strokeWidth="3" strokeLinecap="round" />
              {lineData.points.map((point, index) => (
                <circle
                  key={index}
                  cx={point.x}
                  cy={point.y}
                  r={lineHoverIndex === index ? 5 : 3}
                  fill={lineHoverIndex === index ? '#D1495B' : '#1B4332'}
                  opacity={lineHoverIndex === null || lineHoverIndex === index ? 1 : 0.35}
                />
              ))}
              {monthTicks.map((tick) => {
                const width = 600;
                const pad = 24;
                const x = pad + (tick.index / Math.max(daily.length - 1, 1)) * (width - pad * 2);
                return (
                  <text
                    key={tick.label + tick.index}
                    x={x}
                    y="210"
                    className="chart-label"
                    textAnchor="middle"
                  >
                    {tick.label}
                  </text>
                );
              })}
            </svg>
          )}
        </div>

        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Cashflow rhythm</div>
              <div className="panel-sub">Income up, expenses down</div>
            </div>
            <div className="panel-pill">
              {barHoverIndex !== null && buckets[barHoverIndex]
                ? `${formatDate(buckets[barHoverIndex].label)} · +£${buckets[barHoverIndex].income.toFixed(2)} / -£${buckets[barHoverIndex].expenses.toFixed(2)}`
                : 'Grouped'}
            </div>
          </div>
          {buckets.length === 0 ? (
            <div className="empty">Waiting for transactions.</div>
          ) : (
            <svg
              viewBox="0 0 600 220"
              className="chart-svg chart-hover"
              onMouseLeave={() => setBarHoverIndex(null)}
              onMouseMove={(event) => {
                const rect = (event.currentTarget as SVGSVGElement).getBoundingClientRect();
                const width = 600;
                const pad = 24;
                const ratio = (event.clientX - rect.left) / rect.width;
                const x = ratio * width;
                const barSpace = (width - pad * 2) / buckets.length;
                const index = Math.floor((x - pad) / barSpace);
                if (index >= 0 && index < buckets.length) {
                  setBarHoverIndex(index);
                }
              }}
            >
              {(() => {
                const width = 600;
                const height = 220;
                const pad = 24;
                const mid = height / 2;
                const maxAbs = Math.max(
                  ...buckets.map((b) => Math.max(b.income, b.expenses, Math.abs(b.net))),
                  1
                );
                const scale = (height / 2 - pad) / maxAbs;
                const barSpace = (width - pad * 2) / buckets.length;
                return buckets.map((bucket, index) => {
                  const x = pad + index * barSpace;
                  const incomeHeight = bucket.income * scale;
                  const expenseHeight = bucket.expenses * scale;
                  const isActive = barHoverIndex === null || barHoverIndex === index;
                  return (
                    <g key={bucket.label}>
                      <rect
                        x={x + barSpace * 0.15}
                        y={mid - incomeHeight}
                        width={barSpace * 0.3}
                        height={incomeHeight}
                        fill={isActive ? '#40916C' : '#9BC1B6'}
                        rx="3"
                      />
                      <rect
                        x={x + barSpace * 0.55}
                        y={mid}
                        width={barSpace * 0.3}
                        height={expenseHeight}
                        fill={isActive ? '#D1495B' : '#E6A1AB'}
                        rx="3"
                      />
                    </g>
                  );
                });
              })()}
              <line x1="24" x2="576" y1="110" y2="110" stroke="#DAD2BC" strokeWidth="1" />
              {bucketMonthTicks.map((tick) => {
                const width = 600;
                const pad = 24;
                const barSpace = (width - pad * 2) / buckets.length;
                const x = pad + tick.index * barSpace + barSpace * 0.1;
                return (
                  <text
                    key={tick.label + tick.index}
                    x={x}
                    y="210"
                    className="chart-label"
                    textAnchor="middle"
                  >
                    {tick.label}
                  </text>
                );
              })}
            </svg>
          )}
        </div>
      </div>

      <div className="grid-secondary">
        <div className="panel chart-card">
          <div className="panel-header">
            <div>
              <div className="panel-title">Expense mix</div>
              <div className="panel-sub">Top categories by spend</div>
            </div>
            <div className="panel-pill">Top 5</div>
          </div>
          {donut.total === 0 ? (
            <div className="empty">No expenses in range.</div>
          ) : (
            <div className="donut-wrap">
              <div className="donut-chart">
                <svg viewBox="0 0 120 120" className="donut">
                  <circle cx="60" cy="60" r="46" fill="none" stroke="#EDE6DB" strokeWidth="16" />
                  {(() => {
                    const radius = 46;
                    const circumference = 2 * Math.PI * radius;
                    let offset = 0;
                    return donut.segments.map((segment, index) => {
                      const value = segment.expenses;
                      const dash = (value / donut.total) * circumference;
                      const strokeDasharray = `${dash} ${circumference - dash}`;
                      const strokeDashoffset = -offset;
                      offset += dash;
                      return (
                        <circle
                          key={segment.category_name}
                          cx="60"
                          cy="60"
                          r={radius}
                          fill="none"
                          stroke={CHART_COLORS[index % CHART_COLORS.length]}
                          strokeWidth="16"
                          strokeDasharray={strokeDasharray}
                          strokeDashoffset={strokeDashoffset}
                          strokeLinecap="round"
                          transform="rotate(-90 60 60)"
                        />
                      );
                    });
                  })()}
                </svg>
                <div className="donut-center">
                  <div className="donut-label">Spend</div>
                  <div className="donut-value">£{donut.total.toFixed(2)}</div>
                </div>
              </div>
              <div className="donut-legend">
                {donut.segments.map((segment, index) => (
                  <div className="legend-row" key={segment.category_name}>
                    <span
                      className="legend-dot"
                      style={{ background: CHART_COLORS[index % CHART_COLORS.length] }}
                    />
                    <span className="legend-name">{segment.category_name}</span>
                    <span className="legend-value">£{segment.expenses.toFixed(2)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

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
                <div className={`list-amount ${txn.amount < 0 ? 'neg' : 'pos'}`}>
                  {formatCurrency(txn.amount)}
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
                  {stmt.closing_balance !== null ? `£${stmt.closing_balance.toFixed(2)}` : '-'}
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
