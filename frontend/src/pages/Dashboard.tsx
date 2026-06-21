import { lazy, Suspense, useEffect, useState, useCallback } from 'react';
import { api, classifyError } from '../api/client';
import { useTranslation } from '../api/i18n';
import { ErrorState, LoadingState } from '../components/ErrorState';
import TrendChart from './dashboard/TrendChart';
import type { GeoItem, OverviewData, TrendItem, UAItem } from './dashboard/types';

const GeoDistributionMap = lazy(() => import('./GeoDistributionMap'));

export default function Dashboard() {
  const { t } = useTranslation();
  const [overview, setOverview] = useState<OverviewData | null>(null);
  const [trend, setTrend] = useState<TrendItem[]>([]);
  const [geo, setGeo] = useState<GeoItem[]>([]);
  const [uas, setUas] = useState<UAItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [overviewRes, trendRes, geoRes, uasRes] = await Promise.allSettled([
        api.get('/api/admin/analytics/overview'),
        api.get('/api/admin/analytics/trend'),
        api.get('/api/admin/analytics/geo'),
        api.get('/api/admin/analytics/user-agents'),
      ]);

      // If ALL requests failed, show error state
      const allFailed = [overviewRes, trendRes, geoRes, uasRes].every(
        (r) => r.status === 'rejected'
      );
      if (allFailed) {
        const firstReason = (overviewRes as PromiseRejectedResult).reason;
        setError(classifyError(firstReason, t));
        return;
      }

      // Populate successful results, leave failed sections as empty/default
      if (overviewRes.status === 'fulfilled') setOverview(overviewRes.value);
      if (trendRes.status === 'fulfilled') setTrend(trendRes.value || []);
      if (geoRes.status === 'fulfilled') setGeo(geoRes.value || []);
      if (uasRes.status === 'fulfilled') setUas(uasRes.value || []);
    } catch (err) {
      setError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatUptime = (seconds: number) => {
    const d = Math.floor(seconds / (3600 * 24));
    const h = Math.floor((seconds % (3600 * 24)) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (d > 0) return t('time.dhm', { d, h, m });
    if (h > 0) return t('time.hm', { h, m });
    return t('time.m', { m });
  };

  if (loading) {
    return <LoadingState label={t('dash.loading')} />;
  }

  if (error) {
    return (
      <div className="animate-slideup">
        <div className="page-header">
          <div>
            <h1 className="page-title">{t('nav.dashboard')}</h1>
            <div className="page-subtitle">{t('dash.subtitle')}</div>
          </div>
        </div>
        <div className="card">
          <ErrorState message={error} onRetry={fetchData} />
        </div>
      </div>
    );
  }

  return (
    <div className="animate-slideup">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('nav.dashboard')}</h1>
          <div className="page-subtitle">
            {t('dash.subtitle')}
          </div>
        </div>
        {overview && (
          <div style={{ display: 'flex', gap: '16px' }}>
            <span className="badge badge-info" style={{ textTransform: 'none' }}>
              {t('dash.uptime')}: {formatUptime(overview.uptime_seconds)}
            </span>
            <span className={`badge ${overview.geoip_enabled ? 'badge-success' : 'badge-neutral'}`} style={{ textTransform: 'none' }}>
              {t('dash.geoip')}: {overview.geoip_enabled ? t('dash.enabled') : t('dash.disabled')}
            </span>
          </div>
        )}
      </div>

      {overview && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '20px', marginBottom: '32px' }}>
          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
            <div style={{ background: 'var(--primary-glow)', border: '1px solid var(--border-active)', padding: '12px', borderRadius: '10px' }}>
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--primary)" strokeWidth="2.5">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                <polyline points="15 3 21 3 21 9"></polyline>
                <line x1="10" y1="14" x2="21" y2="3"></line>
              </svg>
            </div>
            <div>
              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
                {t('dash.total_pv')}
              </div>
              <div style={{ fontSize: '26px', fontWeight: 800, fontFamily: 'var(--font-title)' }}>{overview.total_pv}</div>
            </div>
          </div>

          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
            <div style={{ background: 'rgba(167, 139, 250, 0.12)', border: '1px solid rgba(167, 139, 250, 0.2)', padding: '12px', borderRadius: '10px' }}>
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#a78bfa" strokeWidth="2.5">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                <circle cx="9" cy="7" r="4"></circle>
                <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
              </svg>
            </div>
            <div>
              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
                {t('dash.total_uv')}
              </div>
              <div style={{ fontSize: '26px', fontWeight: 800, fontFamily: 'var(--font-title)' }}>{overview.total_uv}</div>
            </div>
          </div>

          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
            <div style={{ background: 'rgba(52, 211, 153, 0.12)', border: '1px solid rgba(52, 211, 153, 0.2)', padding: '12px', borderRadius: '10px' }}>
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#34d399" strokeWidth="2.5">
                <line x1="12" y1="1" x2="12" y2="23"></line>
                <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
              </svg>
            </div>
            <div>
              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
                {t('dash.today_pv_uv')}
              </div>
              <div style={{ fontSize: '24px', fontWeight: 800, fontFamily: 'var(--font-title)' }}>
                {overview.today_pv} <span style={{ color: 'var(--text-muted)', fontSize: '16px', fontWeight: 400 }}>/ {overview.today_uv}</span>
              </div>
            </div>
          </div>

          <div className="card" style={{ display: 'flex', alignItems: 'center', gap: '20px' }}>
            <div style={{ background: 'rgba(251, 191, 36, 0.12)', border: '1px solid rgba(251, 191, 36, 0.2)', padding: '12px', borderRadius: '10px' }}>
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#fbbf24" strokeWidth="2.5">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
              </svg>
            </div>
            <div>
              <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
                {t('nav.links')}
              </div>
              <div style={{ fontSize: '24px', fontWeight: 800, fontFamily: 'var(--font-title)' }}>
                {overview.active_links} <span style={{ color: 'var(--text-muted)', fontSize: '16px', fontWeight: 400 }}>/ {overview.total_links} {t('dash.active_suffix')}</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Render SVG Line Chart */}
      <div className="card" style={{ marginBottom: '32px', position: 'relative' }}>
        <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
          {t('dash.traffic_trend_15d')}
        </h3>
        {trend.length > 0 ? (
          <div style={{ width: '100%', height: '240px', position: 'relative' }}>
            <TrendChart trend={trend} />
          </div>
        ) : (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '200px', color: 'var(--text-muted)' }}>
            {t('dash.no_trend_15d')}
          </div>
        )}
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
        {/* Geo Distribution */}
        <div className="card">
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
            {t('dash.top_countries')}
          </h3>
          {/* World Map Visualization */}
          <div style={{ marginBottom: '20px', width: '100%' }}>
            <Suspense fallback={<div style={{ height: '260px' }} />}>
              <GeoDistributionMap geo={geo} />
            </Suspense>
          </div>
          <div className="table-container" style={{ border: 'none' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t('dash.country')}</th>
                  <th>{t('dash.region_city')}</th>
                  <th style={{ textAlign: 'right' }}>{t('dash.ip_count')}</th>
                  <th style={{ textAlign: 'right' }}>{t('dash.uv_count')}</th>
                  <th style={{ textAlign: 'right' }}>{t('dash.requests')}</th>
                </tr>
              </thead>
              <tbody>
                {geo.length > 0 ? (
                  geo.map((item, idx) => (
                    <tr key={idx}>
                      <td>{item.country || t('dash.unknown')}</td>
                      <td>{formatRegionCity(item)}</td>
                      <td style={{ textAlign: 'right', fontWeight: 600 }}>{item.ip_count}</td>
                      <td style={{ textAlign: 'right', fontWeight: 600 }}>{item.uv}</td>
                      <td style={{ textAlign: 'right', fontWeight: 600 }}>{item.requests}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '24px 0' }}>
                      {t('dash.no_geo')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* User-Agents and Cache status */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
          {overview && (
            <div className="card" style={{ flex: 1 }}>
              <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
                {t('dash.cache_performance')}
              </h3>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '24px' }}>
                <div style={{ background: 'rgba(255,255,255,0.02)', padding: '16px', borderRadius: '8px', border: '1px solid var(--border-glass)' }}>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t('dash.cache_usage')}</div>
                  <div style={{ fontSize: '18px', fontWeight: 700, color: 'var(--primary)', marginTop: '4px' }}>
                    {overview.cache_objects} <span style={{ fontSize: '12px', color: 'var(--text-muted)', fontWeight: 400 }}>{t('dash.cache_objects_suffix')}</span>
                  </div>
                  <div style={{ fontSize: '12px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    {formatBytes(overview.cache_used_bytes)} / {formatBytes(overview.cache_max_bytes)}
                  </div>
                </div>
                <div style={{ background: 'rgba(255,255,255,0.02)', padding: '16px', borderRadius: '8px', border: '1px solid var(--border-glass)' }}>
                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{t('dash.cache_hit_rate')}</div>
                  <div style={{ fontSize: '24px', fontWeight: 800, color: '#34d399', marginTop: '4px', fontFamily: 'var(--font-title)' }}>
                    {overview.cache_hit_rate.toFixed(1)}%
                  </div>
                </div>
              </div>
            </div>
          )}

          <div className="card" style={{ flex: 2 }}>
            <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
              {t('dash.top_ua')}
            </h3>
            <div className="table-container" style={{ border: 'none' }}>
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('dash.ua')}</th>
                    <th style={{ textAlign: 'right' }}>{t('dash.requests')}</th>
                  </tr>
                </thead>
                <tbody>
                  {uas.length > 0 ? (
                    uas.map((item, idx) => (
                      <tr key={idx}>
                        <td style={{ maxWidth: '280px', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }} title={item.user_agent}>
                          {item.user_agent || t('dash.unknown')}
                        </td>
                        <td style={{ textAlign: 'right', fontWeight: 600 }}>{item.pv}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={2} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '24px 0' }}>
                        {t('dash.no_clients')}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function formatRegionCity(item: GeoItem) {
  const region = item.region || '';
  const city = item.city || '';
  if (region && city && region !== city) return `${region} / ${city}`;
  return region || city || '-';
}
