import { useEffect, useState } from 'react';
import { api, getErrorMessage, classifyError } from '../api/client';
import { useTranslation } from '../api/i18n';
import { useToast } from '../components/Toast';
import { ErrorState, LoadingState } from '../components/ErrorState';

interface CacheStatus {
  count: number;
  memory_used_bytes: number;
  memory_max_bytes: number;
  hits: number;
  misses: number;
  hit_rate_percent: number;
  enabled: boolean;
}

export default function Cache() {
  const { t } = useTranslation();
  const toast = useToast();
  const [status, setStatus] = useState<CacheStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [clearing, setClearing] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    fetchCacheStatus();
  }, []);

  const fetchCacheStatus = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await api.get('/api/admin/cache/status');
      setStatus(data);
    } catch (err) {
      console.error('Failed to load cache status:', err);
      setLoadError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  };

  const handleClearAll = async () => {
    if (!confirm(t('cache.confirm_clear'))) return;
    setClearing(true);
    try {
      await api.post('/api/admin/cache/clear');
      toast.success(t('cache.success.clear_all'));
      fetchCacheStatus();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('cache.error.clear_failed')));
    } finally {
      setClearing(false);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading) {
    return <LoadingState label={t('cache.loading')} />;
  }

  if (loadError) {
    return (
      <div className="animate-slideup">
        <div className="page-header">
          <div>
            <h1 className="page-title">{t('nav.cache')}</h1>
            <div className="page-subtitle">{t('cache.page_subtitle')}</div>
          </div>
        </div>
        <div className="card">
          <ErrorState message={loadError} onRetry={fetchCacheStatus} />
        </div>
      </div>
    );
  }

  return (
    <div className="animate-slideup">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('nav.cache')}</h1>
          <div className="page-subtitle">
            {t('cache.page_subtitle')}
          </div>
        </div>
        <button className="btn btn-danger" onClick={handleClearAll} disabled={clearing || !status?.count}>
          {t('cache.action.clear_all')}
        </button>
      </div>

      {status && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '24px', marginBottom: '32px' }}>
          {/* Status Card */}
          <div className="card" style={{ display: 'flex', flexDirection: 'column', justifyItems: 'center', gap: '16px' }}>
            <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '16px', color: 'var(--text-secondary)' }}>
              {t('cache.engine_status')}
            </h3>
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <span className={`badge ${status.enabled ? 'badge-success' : 'badge-neutral'}`}>
                {status.enabled ? t('cache.enabled') : t('cache.disabled')}
              </span>
              <span style={{ fontSize: '14px', color: 'var(--text-muted)' }}>
                {t('cache.global_memory')}
              </span>
            </div>
          </div>

          {/* Usage Card */}
          <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
              {t('cache.stat.memory_usage')}
            </div>
            <div style={{ fontSize: '28px', fontWeight: 800, color: 'var(--primary)', fontFamily: 'var(--font-title)' }}>
              {formatBytes(status.memory_used_bytes)}
            </div>
            <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
              {t('cache.stat.limit')}: {formatBytes(status.memory_max_bytes)}
            </div>
          </div>

          {/* Objects Card */}
          <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
              {t('cache.stat.objects')}
            </div>
            <div style={{ fontSize: '28px', fontWeight: 800, color: '#a78bfa', fontFamily: 'var(--font-title)' }}>
              {status.count}
            </div>
            <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
              {t('cache.stat.active_keys')}
            </div>
          </div>

          {/* Hit Rate Card */}
          <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={{ fontSize: '13px', color: 'var(--text-secondary)', fontWeight: 600 }}>
              {t('cache.stat.hit_rate')}
            </div>
            <div style={{ fontSize: '28px', fontWeight: 800, color: '#34d399', fontFamily: 'var(--font-title)' }}>
              {status.hit_rate_percent.toFixed(1)}%
            </div>
            <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>
              {t('cache.stat.hits_misses', { hits: status.hits, misses: status.misses })}
            </div>
          </div>
        </div>
      )}

      <div className="card" style={{ padding: '32px' }}>
        <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '16px' }}>
          {t('cache.about.title')}
        </h3>
        <p style={{ color: 'var(--text-secondary)', lineHeight: 1.6, marginBottom: '16px' }}>
          {t('cache.about.intro')}
        </p>
        <ul style={{ color: 'var(--text-secondary)', paddingLeft: '20px', lineHeight: 1.8, marginBottom: '24px', listStyleType: 'disc' }}>
          <li>
            {t('cache.about.item1')}
          </li>
          <li>
            {t('cache.about.item2')}
          </li>
          <li>
            {t('cache.about.item3')}
          </li>
          <li>
            {t('cache.about.item4')}
          </li>
        </ul>
        <button className="btn btn-secondary" onClick={fetchCacheStatus}>
          {t('cache.action.refresh_metrics')}
        </button>
      </div>
    </div>
  );
}
