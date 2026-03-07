import {
  Bar,
  BarChart,
  CartesianGrid,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import ChartTooltip from '../ChartTooltip';

export type Bucket = {
  label: string;
  income: number;
  expenses: number;
  net: number;
};

const fmtMonth = (dateStr: string) =>
  new Date(dateStr).toLocaleDateString('en-GB', { month: 'short' });

const fmtCurrency = (v: number) => `£${Math.abs(v).toFixed(0)}`;

interface Props {
  buckets: Bucket[];
}

export default function CashflowRhythm({ buckets }: Props) {
  if (buckets.length === 0) {
    return <div className="empty">Waiting for transactions.</div>;
  }

  const data = buckets.map((b) => ({
    label: b.label,
    expenses: b.expenses,
  }));

  const total = data.reduce((sum, d) => sum + d.expenses, 0);
  const avg = total / data.length;

  return (
    <ResponsiveContainer width="100%" height={240}>
      <BarChart data={data} margin={{ top: 8, right: 12, bottom: 4, left: 4 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="var(--line)" />
        <XAxis
          dataKey="label"
          tickFormatter={fmtMonth}
          tick={{ fontSize: 11, fill: 'var(--muted)' }}
          axisLine={{ stroke: 'var(--line)' }}
          tickLine={false}
          minTickGap={40}
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
          labelFormatter={(d) =>
            new Date(d).toLocaleDateString('en-GB', {
              day: '2-digit',
              month: 'short',
              year: 'numeric',
            })
          }
        />
        <ReferenceLine
          y={avg}
          stroke="var(--accent)"
          strokeDasharray="4 4"
          label={{ value: `avg £${avg.toFixed(0)}`, position: 'right', fontSize: 11, fill: 'var(--accent)' }}
        />
        <Bar dataKey="expenses" name="expenses" fill="#D1495B" radius={[3, 3, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
