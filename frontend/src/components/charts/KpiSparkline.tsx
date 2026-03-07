import { Area, AreaChart, ResponsiveContainer } from 'recharts';

interface Props {
  data: number[];
  color: string;
}

export default function KpiSparkline({ data, color }: Props) {
  if (data.length < 2) return null;

  const points = data.map((v, i) => ({ i, v }));

  return (
    <div className="kpi-sparkline">
      <ResponsiveContainer width="100%" height={36}>
        <AreaChart data={points} margin={{ top: 2, right: 0, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id={`spark-${color.replace('#', '')}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity={0.35} />
              <stop offset="100%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="v"
            stroke={color}
            strokeWidth={1.5}
            fill={`url(#spark-${color.replace('#', '')})`}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
