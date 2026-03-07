export type RangePreset = '30d' | '90d' | '6m' | '1y' | 'all' | 'custom';

const PRESETS: { value: RangePreset; label: string }[] = [
  { value: '30d', label: '30d' },
  { value: '90d', label: '90d' },
  { value: '6m', label: '6m' },
  { value: '1y', label: '1y' },
  { value: 'all', label: 'All' },
  { value: 'custom', label: 'Custom' },
];

interface Props {
  preset: RangePreset;
  onPresetChange: (p: RangePreset) => void;
  customStart: string;
  customEnd: string;
  onCustomChange: (start: string, end: string) => void;
}

export default function DateRangePicker({
  preset,
  onPresetChange,
  customStart,
  customEnd,
  onCustomChange,
}: Props) {
  return (
    <div className="date-range-bar">
      <div className="range-pills">
        {PRESETS.map((p) => (
          <button
            key={p.value}
            className={`range-pill${preset === p.value ? ' active' : ''}`}
            onClick={() => onPresetChange(p.value)}
          >
            {p.label}
          </button>
        ))}
      </div>
      {preset === 'custom' && (
        <div className="range-custom-inputs">
          <input
            type="date"
            value={customStart}
            onChange={(e) => onCustomChange(e.target.value, customEnd)}
          />
          <span className="range-sep">to</span>
          <input
            type="date"
            value={customEnd}
            onChange={(e) => onCustomChange(customStart, e.target.value)}
          />
        </div>
      )}
    </div>
  );
}
