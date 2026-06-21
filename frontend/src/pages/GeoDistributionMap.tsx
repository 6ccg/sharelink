import { useMemo, useState } from 'react';
import type { WheelEvent } from 'react';
import { geoMercator, geoPath } from 'd3-geo';
import { useTheme } from '../api/theme';
import { buildMapFeatures } from './dashboard/geoMapData';
import type { GeoItem } from './dashboard/types';
import {
  DEFAULT_VIEW,
  MAP_HEIGHT,
  MAP_WIDTH,
  WORLD_VIEW,
  panView,
  zoomView,
  type DragState,
  type MapView,
} from './dashboard/geoMapView';

interface HoveredFeature {
  name: string;
  type: 'country' | 'province';
  value: number;
  x: number;
  y: number;
}

const projection = geoMercator()
  .scale(130)
  .translate([MAP_WIDTH / 2, MAP_HEIGHT / 2 + 55]);
const path = geoPath(projection);

export default function GeoDistributionMap({ geo }: { geo: GeoItem[] }) {
  const { theme } = useTheme();
  const [hovered, setHovered] = useState<HoveredFeature | null>(null);
  const [view, setView] = useState<MapView>(DEFAULT_VIEW);
  const [drag, setDrag] = useState<DragState | null>(null);

  const { features, maxPV } = useMemo(() => buildMapFeatures(geo), [geo]);
  const emptyFill = theme === 'dark' ? 'rgba(255,255,255,0.04)' : 'rgba(15,23,42,0.05)';
  const emptyStroke = theme === 'dark' ? 'rgba(255,255,255,0.08)' : 'rgba(15,23,42,0.1)';

  return (
    <div style={{ position: 'relative', width: '100%', userSelect: 'none' }}>
      <div
        style={{
          position: 'absolute',
          top: 8,
          right: 8,
          display: 'flex',
          gap: '6px',
          zIndex: 3,
        }}
      >
        <MapControlButton label="+" title="放大" onClick={() => setView((current) => zoomView(current, 0.78))} />
        <MapControlButton label="-" title="缩小" onClick={() => setView((current) => zoomView(current, 1.28))} />
        <MapControlButton label="中国" title="回到中国视角" onClick={() => setView(DEFAULT_VIEW)} />
        <MapControlButton label="全球" title="回到全球视角" onClick={() => setView(WORLD_VIEW)} />
      </div>
      <svg
        viewBox={`${view.x} ${view.y} ${view.width} ${view.height}`}
        role="img"
        aria-label="Geo distribution map"
        style={{
          width: '100%',
          height: 'auto',
          display: 'block',
          cursor: drag ? 'grabbing' : 'grab',
          touchAction: 'none',
        }}
        onWheel={(event) => handleWheel(event, view, setView)}
        onPointerDown={(event) => {
          event.currentTarget.setPointerCapture(event.pointerId);
          setHovered(null);
          setDrag({
            pointerId: event.pointerId,
            startX: event.clientX,
            startY: event.clientY,
            view,
          });
        }}
        onPointerMove={(event) => {
          if (!drag || drag.pointerId !== event.pointerId) return;
          const rect = event.currentTarget.getBoundingClientRect();
          const dx = ((event.clientX - drag.startX) / rect.width) * drag.view.width;
          const dy = ((event.clientY - drag.startY) / rect.height) * drag.view.height;
          setView(panView(drag.view, dx, dy));
        }}
        onPointerUp={(event) => {
          event.currentTarget.releasePointerCapture(event.pointerId);
          setDrag(null);
        }}
        onPointerCancel={() => setDrag(null)}
        onMouseLeave={() => setHovered(null)}
      >
        {features.map((item) => {
          const d = path(item);
          if (!d) return null;
          const props = item.properties;
          const hasValue = props.value > 0;
          const fill = hasValue
            ? getFillColor(props.value, maxPV, props.type, theme)
            : emptyFill;
          return (
            <path
              key={`${props.type}-${props.name}`}
              d={d}
              fill={fill}
              stroke={hasValue ? (theme === 'dark' ? 'rgba(248,250,252,0.22)' : 'rgba(15,23,42,0.18)') : emptyStroke}
              strokeWidth={props.type === 'province' ? 0.45 : 0.35}
              vectorEffect="non-scaling-stroke"
              style={{ cursor: hasValue ? 'pointer' : undefined, transition: 'fill 120ms ease, opacity 120ms ease' }}
              onMouseMove={(event) => {
                if (drag) return;
                if (!hasValue) return;
                const rect = event.currentTarget.ownerSVGElement?.getBoundingClientRect();
                if (!rect) return;
                setHovered({
                  name: props.name,
                  type: props.type,
                  value: props.value,
                  x: event.clientX - rect.left,
                  y: event.clientY - rect.top,
                });
              }}
              onMouseLeave={() => setHovered(null)}
            />
          );
        })}
      </svg>
      {hovered && (
        <div
          style={{
            position: 'absolute',
            left: hovered.x + 12,
            top: hovered.y + 12,
            padding: '8px 10px',
            borderRadius: '8px',
            border: '1px solid var(--border-glass)',
            background: theme === 'dark' ? '#1e293b' : '#ffffff',
            color: theme === 'dark' ? '#f8fafc' : '#0f172a',
            boxShadow: 'var(--shadow-md)',
            fontSize: '12px',
            fontWeight: 700,
            pointerEvents: 'none',
            zIndex: 2,
          }}
        >
          <div>{hovered.name}</div>
          <div style={{ color: 'var(--text-secondary)', marginTop: '2px' }}>
            {hovered.value} PV
          </div>
        </div>
      )}
    </div>
  );
}

function MapControlButton({
  label,
  title,
  onClick,
}: {
  label: string;
  title: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      style={{
        minWidth: label.length > 1 ? '44px' : '30px',
        height: '30px',
        borderRadius: '6px',
        border: '1px solid var(--border-glass)',
        background: 'var(--bg-glass)',
        color: 'var(--text-primary)',
        fontSize: '12px',
        fontWeight: 700,
        cursor: 'pointer',
      }}
    >
      {label}
    </button>
  );
}

function handleWheel(
  event: WheelEvent<SVGSVGElement>,
  view: MapView,
  setView: (updater: (current: MapView) => MapView) => void
) {
  event.preventDefault();
  const rect = event.currentTarget.getBoundingClientRect();
  const pointerX = view.x + (event.clientX - rect.left) / rect.width * view.width;
  const pointerY = view.y + (event.clientY - rect.top) / rect.height * view.height;
  const factor = event.deltaY < 0 ? 0.86 : 1.16;
  setView((current) => zoomView(current, factor, pointerX, pointerY));
}

function getFillColor(value: number, maxValue: number, type: 'country' | 'province', theme: string) {
  const ratio = maxValue > 0 ? Math.sqrt(value / maxValue) : 0;
  const alpha = 0.28 + ratio * 0.68;
  if (type === 'province') {
    return theme === 'dark'
      ? `rgba(56, 189, 248, ${alpha})`
      : `rgba(2, 132, 199, ${alpha})`;
  }
  return theme === 'dark'
    ? `rgba(167, 139, 250, ${alpha})`
    : `rgba(79, 70, 229, ${alpha})`;
}
