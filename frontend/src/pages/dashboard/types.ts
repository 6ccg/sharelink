export interface OverviewData {
  total_links: number;
  active_links: number;
  total_pv: number;
  total_uv: number;
  today_pv: number;
  today_uv: number;
  cache_objects: number;
  cache_used_bytes: number;
  cache_max_bytes: number;
  cache_hit_rate: number;
  geoip_enabled: boolean;
  geoip_db_path: string;
  uptime_seconds: number;
}

export interface TrendItem {
  date: string;
  pv: number;
  uv: number;
}

export interface GeoItem {
  country: string;
  region: string;
  requests: number;
  uv: number;
  ip_count: number;
}

export interface UAItem {
  user_agent: string;
  pv: number;
}
