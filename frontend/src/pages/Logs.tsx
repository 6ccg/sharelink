import { useEffect, useState } from 'react';
import { api, classifyError } from '../api/client';
import { useTranslation } from '../api/i18n';
import { TableErrorRow, TableLoadingRow } from '../components/ErrorState';

interface VisitLog {
  id: number;
  link_id: number | null;
  prefix: string;
  slug: string;
  public_path: string;
  ip: string;
  country: string;
  region: string;
  city: string;
  access_time: string;
  mode: string;
  status: string;
  blocked_reason: string | null;
  response_status_code: number;
  upstream_status_code: number;
  response_size: number;
  cache_status: string;
  user_agent: string;
  referer: string;
}

export default function Logs() {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<VisitLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Filters
  const [ip, setIp] = useState('');
  const [country, setCountry] = useState('');
  const [status, setStatus] = useState('');
  const [cacheStatus, setCacheStatus] = useState('');
  const [keyword, setKeyword] = useState('');

  useEffect(() => {
    fetchLogs();
  }, [page, ip, country, status, cacheStatus, keyword]);

  const fetchLogs = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const queryParams = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
        ip,
        country,
        status,
        cache_status: cacheStatus,
        keyword,
      });
      const data = await api.get(`/api/admin/logs?${queryParams.toString()}`);
      setLogs(data.items || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to load visit logs:', err);
      setLoadError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusBadge = (logItem: VisitLog) => {
    if (logItem.status === 'success') {
      return <span className="badge badge-success">{t('logs.status.success')}</span>;
    }
    if (logItem.status === 'blocked') {
      return (
        <span className="badge badge-error" title={logItem.blocked_reason || 'Blocked'}>
          {t('logs.status.blocked', { reason: logItem.blocked_reason || 'UA' })}
        </span>
      );
    }
    if (logItem.status === 'expired') {
      return <span className="badge badge-warning">{t('logs.status.expired')}</span>;
    }
    return <span className="badge badge-error">{t('logs.status.failed')}</span>;
  };

  const getCacheBadge = (cacheState: string) => {
    if (cacheState === 'hit') return <span className="badge badge-success">{t('logs.cache.hit')}</span>;
    if (cacheState === 'miss') return <span className="badge badge-error">{t('logs.cache.miss')}</span>;
    if (cacheState === 'bypass') return <span className="badge badge-neutral">{t('logs.cache.bypass')}</span>;
    return <span className="badge badge-neutral" style={{ opacity: 0.5 }}>{t('logs.cache.off')}</span>;
  };

  const formatDate = (isoString: string) => {
    const d = new Date(isoString);
    return d.toLocaleString();
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="animate-slideup">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('nav.logs')}</h1>
          <div className="page-subtitle">
            {t('logs.subtitle')}
          </div>
        </div>
        <button className="btn btn-secondary" onClick={() => {
          setIp('');
          setCountry('');
          setStatus('');
          setCacheStatus('');
          setKeyword('');
          setPage(1);
        }}>
          {t('logs.action.reset')}
        </button>
      </div>

      {/* Filters Toolbar */}
      <div className="card" style={{ marginBottom: '24px', padding: '16px' }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '16px' }}>
          <div>
            <input
              type="text"
              placeholder={t('logs.filter.keyword')}
              className="form-input"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value);
                setPage(1);
              }}
            />
          </div>

          <div>
            <input
              type="text"
              placeholder={t('logs.filter.ip_short')}
              className="form-input"
              value={ip}
              onChange={(e) => {
                setIp(e.target.value);
                setPage(1);
              }}
            />
          </div>

          <div>
            <input
              type="text"
              placeholder={t('logs.filter.country')}
              className="form-input"
              value={country}
              onChange={(e) => {
                setCountry(e.target.value);
                setPage(1);
              }}
            />
          </div>

          <div>
            <select
              className="form-select"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t('logs.filter.status')}</option>
              <option value="success">{t('logs.filter.status.success')}</option>
              <option value="blocked">{t('logs.filter.status.blocked')}</option>
              <option value="expired">{t('logs.filter.status.expired')}</option>
              <option value="failed">{t('logs.filter.status.failed')}</option>
            </select>
          </div>

          <div>
            <select
              className="form-select"
              value={cacheStatus}
              onChange={(e) => {
                setCacheStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t('logs.filter.cache.all')}</option>
              <option value="hit">{t('logs.cache.hit')}</option>
              <option value="miss">{t('logs.cache.miss')}</option>
              <option value="bypass">{t('logs.cache.bypass')}</option>
              <option value="disabled">{t('logs.filter.cache.disabled')}</option>
            </select>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div className="table-container">
        <table className="data-table">
          <thead>
            <tr>
              <th>{t('logs.col.time')}</th>
              <th>{t('links.col.path')}</th>
              <th className="cell-center">{t('links.col.mode')}</th>
              <th>{t('logs.col.ip_info')}</th>
              <th>{t('logs.col.response')}</th>
              <th>{t('logs.col.cache')}</th>
              <th>{t('logs.col.client')}</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableLoadingRow colSpan={7} label={t('logs.loading')} />
            ) : loadError ? (
              <TableErrorRow colSpan={7} message={loadError} onRetry={fetchLogs} />
            ) : logs.length > 0 ? (
              logs.map((item) => (
                <tr key={item.id}>
                  <td style={{ whiteSpace: 'nowrap' }}>{formatDate(item.access_time)}</td>
                  <td>
                    <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{item.public_path}</span>
                  </td>
                  <td className="cell-center">
                    <span className={`badge badge-fixed badge-mode ${item.mode === 'proxy' ? 'badge-info' : 'badge-neutral'}`}>
                      {item.mode === 'proxy' ? t('links.mode.proxy') : t('links.mode.redirect')}
                    </span>
                  </td>
                  <td style={{ maxWidth: '180px' }}>
                    <div>{item.ip}</div>
                    <div 
                      style={{ 
                        fontSize: '12px', 
                        color: 'var(--text-secondary)', 
                        marginTop: '2px',
                        textOverflow: 'ellipsis',
                        overflow: 'hidden',
                        whiteSpace: 'nowrap'
                      }} 
                      title={`${item.country || t('logs.unknown_region')} - ${item.region || ''} ${item.city || ''}`}
                    >
                      {item.country || t('logs.unknown_region')} - {item.region || ''} {item.city || ''}
                    </div>
                  </td>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      {getStatusBadge(item)}
                      {item.status === 'success' && (
                        <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-secondary)' }}>
                          {item.response_status_code}
                        </span>
                      )}
                    </div>
                    {item.response_size > 0 && (
                      <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginTop: '2px' }}>
                        {t('logs.size')}: {formatSize(item.response_size)}
                      </div>
                    )}
                  </td>
                  <td>{getCacheBadge(item.cache_status)}</td>
                  <td style={{ maxWidth: '300px' }}>
                    <div style={{ fontSize: '12px', color: 'var(--text-secondary)', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }} title={item.user_agent}>
                      UA: {item.user_agent || '-'}
                    </div>
                    {item.referer && (
                      <div style={{ fontSize: '11px', color: 'var(--text-muted)', textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', marginTop: '2px' }} title={item.referer}>
                        Ref: {item.referer}
                      </div>
                    )}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={7} style={{ textAlign: 'center', padding: '40px 0', color: 'var(--text-muted)' }}>
                  {t('logs.empty')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="pagination">
          <button
            className={`pagination-btn ${page === 1 ? 'disabled' : ''}`}
            onClick={() => page > 1 && setPage(page - 1)}
            disabled={page === 1}
          >
            &lt;
          </button>
          {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
            <button
              key={p}
              className={`pagination-btn ${page === p ? 'active' : ''}`}
              onClick={() => setPage(p)}
            >
              {p}
            </button>
          ))}
          <button
            className={`pagination-btn ${page === totalPages ? 'disabled' : ''}`}
            onClick={() => page < totalPages && setPage(page + 1)}
            disabled={page === totalPages}
          >
            &gt;
          </button>
        </div>
      )}
    </div>
  );
}
