import { useMemo } from 'react';
import {
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { Category, MonthlySpendPoint } from '../../api/client';
import ChartTooltip from '../ChartTooltip';

interface Props {
  series: MonthlySpendPoint[];
  categories: Category[];
  selectedCategory: number | 'all';
  onCategoryChange: (value: number | 'all') => void;
}

export default function MonthlySpend({
  series,
  categories,
  selectedCategory,
  onCategoryChange,
}: Props) {
  // 2-month moving average trend line
  const data = useMemo(() => {
    return series.map((point, i) => {
      let trend: number | null = null;
      if (i >= 1) {
        trend = (series[i].total_expenses + series[i - 1].total_expenses) / 2;
      }
      return {
        month: point.month,
        total_expenses: point.total_expenses,
        trend,
      };
    });
  }, [series]);

  const fmtMonth = (m: string) => {
    const parts = m.split('-');
    const monthNames = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    return monthNames[parseInt(parts[1], 10) - 1] ?? m;
  };

  return (
    <>
      <div className="monthly-controls">
        <label>
          Category
          <select
            className="table-select"
            value={selectedCategory === 'all' ? 'all' : selectedCategory}
            onChange={(e) => {
              const v = e.target.value;
              onCategoryChange(v === 'all' ? 'all' : Number(v));
            }}
          >
            <option value="all">All expenses</option>
            {categories
              .filter((cat) => cat.is_expense)
              .map((cat) => (
                <option key={cat.id} value={cat.id}>
                  {cat.name}
                </option>
              ))}
          </select>
        </label>
      </div>
      {data.length === 0 ? (
        <div className="empty">No monthly data yet.</div>
      ) : (
        <ResponsiveContainer width="100%" height={220}>
          <ComposedChart data={data} margin={{ top: 8, right: 12, bottom: 4, left: 4 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--line)" />
            <XAxis
              dataKey="month"
              tickFormatter={fmtMonth}
              tick={{ fontSize: 11, fill: 'var(--muted)' }}
              axisLine={{ stroke: 'var(--line)' }}
              tickLine={false}
            />
            <YAxis
              tickFormatter={(v: number) => `£${v.toFixed(0)}`}
              tick={{ fontSize: 11, fill: 'var(--muted)' }}
              axisLine={false}
              tickLine={false}
              width={52}
            />
            <Tooltip content={<ChartTooltip />} labelFormatter={(label) => fmtMonth(String(label))} />
            <Bar
              dataKey="total_expenses"
              name="expenses"
              fill="#3B8B80"
              radius={[4, 4, 0, 0]}
              barSize={28}
            />
            <Line
              type="monotone"
              dataKey="trend"
              name="trend"
              stroke="#E09F3E"
              strokeWidth={2}
              strokeDasharray="6 3"
              dot={false}
              connectNulls
            />
          </ComposedChart>
        </ResponsiveContainer>
      )}
    </>
  );
}
