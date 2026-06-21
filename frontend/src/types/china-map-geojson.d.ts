declare module 'china-map-geojson' {
  import type { FeatureCollection, Geometry } from 'geojson';

  export const ChinaData: FeatureCollection<Geometry, { name: string }>;
}
