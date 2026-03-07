import { useEffect, useState } from 'react';
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { getSpendingPace, type SpendingPaceResponse } from '../../api/client';
import ChartTooltip from '../ChartTooltip';

const fmtCurrency = (v: number) => `£${v.toFixed(0)}`;

export default function BalanceTrajectory() {
  const [pace, setPace] = useState<SpendingPaceResponse | null>(null);

  useEffect(() => {
    getSpendingPace().then(setPace).catch(console.error);
  }, []);

  if (!pace || pace.current_series.length === 0) {
    return <div className="empty">No data yet. Import a statement to see spending.</div>;
  }

  // Merge current and previous month by day number
  const prevMap = new Map(pace.prev_series.map((p) => [p.day, p.cumulative]));
  const data = pace.current_series.map((p) => ({
    day: p.day,
    current: p.cumulative,
    previous: prevMap.get(p.day) ?? null,
  }));

  return (
    <ResponsiveContainer width="100%" height={240}>
      <AreaChart data={data} margin={{ top: 8, right: 12, bottom: 4, left: 4 }}>
        <defs>
          <linearGradient id="spendGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#D1495B" stopOpacity={0.25} />
            <stop offset="100%" stopColor="#D1495B" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--line)" />
        <XAxis
          dataKey="day"
          tick={{ fontSize: 11, fill: 'var(--muted)' }}
          axisLine={{ stroke: 'var(--line)' }}
          tickLine={false}
        />
        <YAxis
          tickFormatter={fmtCurrency}
          tick={{ fontSize: 11, fill: 'var(--muted)' }}
          axisLine={false}
          tickLine={false}
          width={52}
        />
        <Tooltip
          content={<ChartTooltip />}
          labelFormatter={(day) => `Day ${day}`}
        />
        <Line
          type="monotone"
          dataKey="previous"
          name="Last month"
          stroke="var(--muted)"
          strokeWidth={1.5}
          strokeDasharray="4 4"
          dot={false}
          connectNulls
        />
        <Area
          type="monotone"
          dataKey="current"
          name="This month"
          stroke="#D1495B"
          strokeWidth={2.5}
          fill="url(#spendGrad)"
          dot={false}
          activeDot={{ r: 5, fill: '#D1495B', stroke: '#fff', strokeWidth: 2 }}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}
