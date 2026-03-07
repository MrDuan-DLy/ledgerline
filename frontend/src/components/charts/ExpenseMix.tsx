import { useState } from 'react';
import { Cell, Pie, PieChart, ResponsiveContainer } from 'recharts';
import type { CategoryTotal } from '../../api/client';

const CHART_COLORS = ['#2D6A4F', '#3B8B80', '#E09F3E', '#D1495B', '#6D4C41', '#7A6C5D'];

interface Props {
  categories: CategoryTotal[];
  total: number;
}

export default function ExpenseMix({ categories, total }: Props) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  if (total === 0) {
    return <div className="empty">No expenses in range.</div>;
  }

  return (
    <div className="donut-wrap">
      <div className="donut-chart">
        <ResponsiveContainer width={120} height={120}>
          <PieChart>
            <Pie
              data={categories}
              dataKey="expenses"
              nameKey="category_name"
              cx="50%"
              cy="50%"
              innerRadius={38}
              outerRadius={activeIndex !== null ? 56 : 54}
              onMouseLeave={() => setActiveIndex(null)}
            >
              {categories.map((_, i) => (
                <Cell
                  key={i}
                  fill={CHART_COLORS[i % CHART_COLORS.length]}
                  stroke={activeIndex === i ? CHART_COLORS[i % CHART_COLORS.length] : 'none'}
                  strokeWidth={activeIndex === i ? 2 : 0}
                  onMouseEnter={() => setActiveIndex(i)}
                />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        <div className="donut-center">
          <div className="donut-label">Spend</div>
          <div className="donut-value">£{total.toFixed(2)}</div>
        </div>
      </div>
      <div className="donut-legend">
        {categories.map((seg, i) => (
          <div
            className={`legend-row${activeIndex === i ? ' legend-active' : ''}`}
            key={seg.category_name}
            onMouseEnter={() => setActiveIndex(i)}
            onMouseLeave={() => setActiveIndex(null)}
          >
            <span
              className="legend-dot"
              style={{ background: CHART_COLORS[i % CHART_COLORS.length] }}
            />
            <span className="legend-name">{seg.category_name}</span>
            <span className="legend-value">£{seg.expenses.toFixed(2)}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
