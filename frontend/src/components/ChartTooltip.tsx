import type { NameType, Payload, ValueType } from 'recharts/types/component/DefaultTooltipContent';

const fmt = (v: number) => `£${Math.abs(v).toFixed(2)}`;

interface Props {
  active?: boolean;
  payload?: ReadonlyArray<Payload<ValueType, NameType>>;
  label?: string | number;
}

export default function ChartTooltip({ active, payload, label }: Props) {
  if (!active || !payload?.length) return null;

  return (
    <div className="chart-tooltip">
      {label != null && <div className="chart-tooltip-label">{String(label)}</div>}
      {payload.map((entry: Payload<ValueType, NameType>, i: number) => {
        const value = Number(entry.value ?? 0);
        const isIncome =
          entry.dataKey === 'income' || entry.name === 'income';
        const isExpense =
          entry.dataKey === 'expenses' ||
          entry.dataKey === 'total_expenses' ||
          entry.name === 'expenses';
        const colorClass = isIncome
          ? 'tooltip-green'
          : isExpense
            ? 'tooltip-rose'
            : '';

        return (
          <div className="chart-tooltip-row" key={String(entry.dataKey ?? entry.name ?? i)}>
            {entry.color && (
              <span
                className="legend-dot"
                style={{ background: entry.color }}
              />
            )}
            <span className="chart-tooltip-name">
              {String(entry.name ?? entry.dataKey)}
            </span>
            <span className={`chart-tooltip-value ${colorClass}`}>
              {fmt(value)}
            </span>
          </div>
        );
      })}
    </div>
  );
}
