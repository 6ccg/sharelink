import { useTranslation } from '../../api/i18n';
import type { TrendItem } from './types';

export default function TrendChart({ trend }: { trend: TrendItem[] }) {
  const { t } = useTranslation();
  const maxPV = Math.max(...trend.map((d) => d.pv), 10);
  const maxUV = Math.max(...trend.map((d) => d.uv), 5);
  const maxVal = Math.max(maxPV, maxUV) * 1.1;

  const width = 1000;
  const height = 200;
  const paddingLeft = 40;
  const paddingRight = 20;
  const paddingTop = 10;
  const paddingBottom = 30;

  const chartWidth = width - paddingLeft - paddingRight;
  const chartHeight = height - paddingTop - paddingBottom;
  const stepX = chartWidth / (trend.length - 1 || 1);
  const getX = (index: number) => paddingLeft + index * stepX;
  const getY = (value: number) => height - paddingBottom - (value / maxVal) * chartHeight;

  const pvPoints = trend.map((d, i) => `${getX(i)},${getY(d.pv)}`).join(' ');
  const uvPoints = trend.map((d, i) => `${getX(i)},${getY(d.uv)}`).join(' ');
  const pvAreaPoints = `${getX(0)},${height - paddingBottom} ${pvPoints} ${getX(trend.length - 1)},${height - paddingBottom}`;
  const uvAreaPoints = `${getX(0)},${height - paddingBottom} ${uvPoints} ${getX(trend.length - 1)},${height - paddingBottom}`;

  return (
    <svg viewBox={`0 0 ${width} ${height}`} style={{ width: '100%', height: '100%', overflow: 'visible' }}>
      <defs>
        <linearGradient id="pvGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#38bdf8" stopOpacity="0.3" />
          <stop offset="100%" stopColor="#38bdf8" stopOpacity="0" />
        </linearGradient>
        <linearGradient id="uvGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#a78bfa" stopOpacity="0.3" />
          <stop offset="100%" stopColor="#a78bfa" stopOpacity="0" />
        </linearGradient>
      </defs>

      {[0, 0.25, 0.5, 0.75, 1].map((ratio, idx) => {
        const y = paddingTop + ratio * chartHeight;
        const val = Math.round(maxVal * (1 - ratio));
        return (
          <g key={idx}>
            <line x1={paddingLeft} y1={y} x2={width - paddingRight} y2={y} stroke="var(--border-glass)" strokeDasharray="4 4" />
            <text x={paddingLeft - 8} y={y + 4} fill="var(--text-muted)" fontSize="11" textAnchor="end">{val}</text>
          </g>
        );
      })}

      <polygon points={pvAreaPoints} fill="url(#pvGrad)" />
      <polygon points={uvAreaPoints} fill="url(#uvGrad)" />
      <polyline points={pvPoints} fill="none" stroke="#38bdf8" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
      <polyline points={uvPoints} fill="none" stroke="#a78bfa" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />

      {trend.map((item, idx) => {
        const x = getX(idx);
        const showLabel = idx % 2 === 0 || idx === trend.length - 1;
        return (
          <g key={idx}>
            <circle cx={x} cy={getY(item.pv)} r="4" fill="#38bdf8" stroke="var(--bg-primary)" strokeWidth="1.5" />
            <circle cx={x} cy={getY(item.uv)} r="4" fill="#a78bfa" stroke="var(--bg-primary)" strokeWidth="1.5" />
            {showLabel && (
              <text x={x} y={height - 8} fill="var(--text-muted)" fontSize="11" textAnchor="middle">
                {item.date.substring(5)}
              </text>
            )}
          </g>
        );
      })}

      <g transform={`translate(${width - 240}, 0)`}>
        <rect x="0" y="0" width="12" height="12" fill="#38bdf8" rx="2" />
        <text x="18" y="10" fill="var(--text-secondary)" fontSize="12" fontWeight="600">
          {t('dash.chart.pv')}
        </text>
        <rect x="120" y="0" width="12" height="12" fill="#a78bfa" rx="2" />
        <text x="138" y="10" fill="var(--text-secondary)" fontSize="12" fontWeight="600">
          {t('dash.chart.uv')}
        </text>
      </g>
    </svg>
  );
}
