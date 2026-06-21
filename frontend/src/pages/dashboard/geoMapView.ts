export interface MapView {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface DragState {
  pointerId: number;
  startX: number;
  startY: number;
  view: MapView;
}

export const MAP_WIDTH = 960;
export const MAP_HEIGHT = 460;
export const WORLD_VIEW: MapView = { x: 0, y: 0, width: MAP_WIDTH, height: MAP_HEIGHT };
export const DEFAULT_VIEW: MapView = { x: 585, y: 125, width: 245, height: 117.4 };

const MIN_VIEW_WIDTH = 120;
const MAX_VIEW_WIDTH = MAP_WIDTH;

export function zoomView(
  view: MapView,
  factor: number,
  pivotX = view.x + view.width / 2,
  pivotY = view.y + view.height / 2
) {
  const nextWidth = Math.min(MAX_VIEW_WIDTH, Math.max(MIN_VIEW_WIDTH, view.width * factor));
  const nextHeight = nextWidth * (MAP_HEIGHT / MAP_WIDTH);
  const ratioX = (pivotX - view.x) / view.width;
  const ratioY = (pivotY - view.y) / view.height;
  return clampView({
    x: pivotX - ratioX * nextWidth,
    y: pivotY - ratioY * nextHeight,
    width: nextWidth,
    height: nextHeight,
  });
}

export function panView(view: MapView, dx: number, dy: number) {
  return clampView({
    ...view,
    x: view.x - dx,
    y: view.y - dy,
  });
}

function clampView(view: MapView): MapView {
  const width = Math.min(MAX_VIEW_WIDTH, Math.max(MIN_VIEW_WIDTH, view.width));
  const height = width * (MAP_HEIGHT / MAP_WIDTH);
  const maxX = MAP_WIDTH - width;
  const maxY = MAP_HEIGHT - height;
  return {
    x: Math.min(Math.max(view.x, 0), Math.max(maxX, 0)),
    y: Math.min(Math.max(view.y, 0), Math.max(maxY, 0)),
    width,
    height,
  };
}
