import { useCallback, useEffect, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type {
  Merchant,
  MerchantTransaction,
  MerchantTransactionsResponse,
} from '../../api/client';
import { getMerchants, getMerchantTransactions, backfillMerchantNames } from '../../api/client';
import ChartTooltip from '../ChartTooltip';

const fmtDate = (d: string) =>
  new Date(d).toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
  });

const fmtDateFull = (d: string) =>
  new Date(d).toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });

interface Props {
  rangeStart: string;
  rangeEnd: string;
}

export default function MerchantDrilldown({ rangeStart, rangeEnd }: Props) {
  const [merchants, setMerchants] = useState<Merchant[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [data, setData] = useState<MerchantTransactionsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [backfilling, setBackfilling] = useState(false);
  const [backfillResult, setBackfillResult] = useState<string | null>(null);

  // Load merchants list once (with transaction counts, filter to non-empty)
  useEffect(() => {
    getMerchants(true).then((list) => {
      const withTxns = list.filter((m) => (m.transaction_count ?? 0) > 0);
      setMerchants(withTxns);
      if (withTxns.length > 0 && selectedId === null) {
        setSelectedId(withTxns[0].id);
      }
    });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Load transactions when merchant or date range changes
  useEffect(() => {
    if (selectedId === null) return;
    setLoading(true);
    getMerchantTransactions(selectedId, {
      start_date: rangeStart,
      end_date: rangeEnd,
    })
      .then(setData)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [selectedId, rangeStart, rangeEnd]);

  const onSelect = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      setSelectedId(Number(e.target.value));
    },
    [],
  );

  const onBackfill = useCallback(async () => {
    setBackfilling(true);
    setBackfillResult(null);
    try {
      const result = await backfillMerchantNames();
      setBackfillResult(result.message);
      // Reload current merchant data to reflect changes
      if (selectedId !== null) {
        const fresh = await getMerchantTransactions(selectedId, {
          start_date: rangeStart,
          end_date: rangeEnd,
        });
        setData(fresh);
      }
    } catch (e) {
      setBackfillResult('Backfill failed.');
      console.error(e);
    }
    setBackfilling(false);
  }, [selectedId, rangeStart, rangeEnd]);

  // Aggregate by date for chart (multiple transactions on same day → summed)
  const chartData = (() => {
    if (!data?.transactions.length) return [];
    const byDate = new Map<string, number>();
    for (const txn of data.transactions) {
      const abs = Math.abs(txn.amount);
      byDate.set(txn.date, (byDate.get(txn.date) ?? 0) + abs);
    }
    return Array.from(byDate.entries())
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([date, spend]) => ({ date, spend: Math.round(spend * 100) / 100 }));
  })();

  return (
    <div className="panel chart-card merchant-drilldown">
      <div className="panel-header">
        <div>
          <div className="panel-title">Merchant spend</div>
          <div className="panel-sub">All transactions for a single merchant</div>
        </div>
        <div className="panel-pill">
          {loading ? 'Loading' : data ? `${data.count} txns` : '-'}
        </div>
      </div>

      <div className="merchant-controls">
        <label>
          Merchant
          <select
            className="table-select"
            value={selectedId ?? ''}
            onChange={onSelect}
          >
            {merchants.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </select>
        </label>
        {data && (
          <div className="merchant-total">
            Total: <strong>£{data.total_spend.toFixed(2)}</strong>
          </div>
        )}
        <button
          className="btn-secondary merchant-backfill-btn"
          onClick={onBackfill}
          disabled={backfilling}
        >
          {backfilling ? 'Running...' : 'Sync names'}
        </button>
      </div>
      {backfillResult && (
        <div className="merchant-backfill-result">{backfillResult}</div>
      )}

      {chartData.length > 0 ? (
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={chartData} margin={{ top: 8, right: 12, bottom: 4, left: 4 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--line)" />
            <XAxis
              dataKey="date"
              tickFormatter={fmtDate}
              tick={{ fontSize: 11, fill: 'var(--muted)' }}
              axisLine={{ stroke: 'var(--line)' }}
              tickLine={false}
              minTickGap={30}
            />
            <YAxis
              tickFormatter={(v: number) => `£${v.toFixed(0)}`}
              tick={{ fontSize: 11, fill: 'var(--muted)' }}
              axisLine={false}
              tickLine={false}
              width={48}
            />
            <Tooltip
              content={<ChartTooltip />}
              labelFormatter={(d) => fmtDateFull(String(d))}
            />
            <Bar
              dataKey="spend"
              name="expenses"
              fill="#3B8B80"
              radius={[3, 3, 0, 0]}
              maxBarSize={24}
            />
          </BarChart>
        </ResponsiveContainer>
      ) : (
        !loading && <div className="empty">No matching transactions found.</div>
      )}

      {data && data.transactions.length > 0 && (
        <div className="merchant-txn-list">
          {data.transactions.slice(0, 20).map((txn: MerchantTransaction) => (
            <div className="list-row" key={txn.id}>
              <div>
                <div className="list-title">{txn.description}</div>
                <div className="list-sub">
                  {fmtDateFull(txn.date)}
                  {txn.category_name && ` · ${txn.category_name}`}
                </div>
              </div>
              <div className={`list-amount ${txn.amount < 0 ? 'neg' : 'pos'}`}>
                {txn.amount < 0 ? '-' : ''}£{Math.abs(txn.amount).toFixed(2)}
              </div>
            </div>
          ))}
          {data.transactions.length > 20 && (
            <div className="empty">
              Showing 20 of {data.transactions.length} transactions.
            </div>
          )}
        </div>
      )}
    </div>
  );
}
