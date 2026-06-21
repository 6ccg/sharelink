import { useEffect, useState } from 'react';
import { api, getErrorMessage, classifyError } from '../api/client';
import { useLocation } from '../api/router';
import { useTranslation } from '../api/i18n';
import { useToast } from '../components/Toast';
import { TableErrorRow, TableLoadingRow } from '../components/ErrorState';

interface ShortLink {
  id: number;
  prefix: string;
  slug: string;
  public_path: string;
  target_url: string;
  mode: string;
  enabled: boolean;
  start_time: string | null;
  expire_time: string | null;
  cache_enabled: boolean;
  cache_ttl: number;
  cache_max_object_size_mb: number;
  filename_mode: string;
  custom_filename: string | null;
  ua_policy_id: number | null;
  note: string | null;
  created_at: string;
}

export default function Links() {
  const { t } = useTranslation();
  const toast = useToast();
  const [links, setLinks] = useState<ShortLink[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [keyword, setKeyword] = useState('');
  const [mode, setMode] = useState('');
  const [enabled, setEnabled] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [, navigate] = useLocation();

  useEffect(() => {
    fetchLinks();
  }, [page, keyword, mode, enabled]);

  const fetchLinks = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const queryParams = new URLSearchParams({
        page: page.toString(),
        page_size: pageSize.toString(),
        keyword,
        mode,
        enabled,
      });
      const data = await api.get(`/api/admin/links?${queryParams.toString()}`);
      setLinks(data.items || []);
      setTotal(data.total || 0);
    } catch (err) {
      console.error('Failed to load links:', err);
      setLoadError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  };

  const handleToggleEnable = async (link: ShortLink) => {
    const endpoint = link.enabled ? `/api/admin/links/${link.id}/disable` : `/api/admin/links/${link.id}/enable`;
    try {
      const updated = await api.post(endpoint);
      setLinks(links.map((l) => (l.id === link.id ? updated : l)));
      toast.success(t('links.success.status_updated'));
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('links.error.operation_failed')));
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm(t('links.confirm_delete'))) return;
    try {
      await api.delete(`/api/admin/links/${id}`);
      toast.success(t('links.success.deleted'));
      fetchLinks();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('links.error.delete_failed')));
    }
  };

  const handleClearLinkCache = async (id: number) => {
    try {
      await api.post(`/api/admin/cache/clear-link/${id}`);
      toast.success(t('links.success.cache_cleared'));
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('links.error.clear_cache_failed')));
    }
  };

  const handleCopy = (publicPath: string) => {
    const fullURL = window.location.origin + publicPath;
    navigator.clipboard.writeText(fullURL).then(
      () => toast.success(t('links.copy_success')),
      () => toast.error(t('links.copy_failed'))
    );
  };

  const totalPages = Math.ceil(total / pageSize);

  return (
    <div className="animate-slideup">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('links.title')}</h1>
          <div className="page-subtitle">
            {t('links.subtitle')}
          </div>
        </div>
        <button className="btn btn-primary" onClick={() => navigate('/admin/links/new')}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          {t('links.create')}
        </button>
      </div>

      {/* Filter and Search Panel */}
      <div className="card" style={{ marginBottom: '24px', padding: '16px' }}>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '16px', alignItems: 'center' }}>
          <div style={{ flex: 1, minWidth: '240px' }}>
            <input
              type="text"
              placeholder={t('links.search')}
              className="form-input"
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value);
                setPage(1);
              }}
            />
          </div>

          <div style={{ width: '120px' }}>
            <select
              className="form-select"
              value={mode}
              onChange={(e) => {
                setMode(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t('links.filter.all_modes')}</option>
              <option value="proxy">{t('links.mode.proxy')}</option>
              <option value="redirect">{t('links.mode.redirect')}</option>
            </select>
          </div>

          <div style={{ width: '120px' }}>
            <select
              className="form-select"
              value={enabled}
              onChange={(e) => {
                setEnabled(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{t('links.filter.all_status')}</option>
              <option value="true">{t('dash.enabled')}</option>
              <option value="false">{t('dash.disabled')}</option>
            </select>
          </div>
        </div>
      </div>

      {/* Data Table */}
      <div className="table-container">
        <table className="data-table links-table">
          <colgroup>
            <col className="links-col-path" />
            <col className="links-col-target" />
            <col className="links-col-mode" />
            <col className="links-col-status" />
            <col className="links-col-cache" />
            <col className="links-col-actions" />
          </colgroup>
          <thead>
            <tr>
              <th>{t('links.col.path')}</th>
              <th>{t('links.col.target')}</th>
              <th className="cell-center">{t('links.col.mode')}</th>
              <th className="cell-center">{t('links.col.status')}</th>
              <th className="cell-center">{t('links.cache')}</th>
              <th className="action-cell">{t('links.col.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <TableLoadingRow colSpan={6} label={t('links.loading')} />
            ) : loadError ? (
              <TableErrorRow colSpan={6} message={loadError} onRetry={fetchLinks} />
            ) : links.length > 0 ? (
              links.map((link) => (
                <tr key={link.id}>
                  <td>
                    <div className="link-path-cell">
                      <span className="link-public-path">
                        {link.public_path}
                      </span>
                      <button
                        className="btn btn-secondary btn-sm icon-copy-btn"
                        onClick={() => handleCopy(link.public_path)}
                        title={t('links.copy_title')}
                      >
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                        </svg>
                      </button>
                    </div>
                    {link.note && (
                      <div className="link-note">
                        {link.note}
                      </div>
                    )}
                  </td>
                  <td className="link-target-cell" title={link.target_url}>
                    {link.target_url}
                  </td>
                  <td className="cell-center">
                    <span className={`badge badge-fixed badge-mode ${link.mode === 'proxy' ? 'badge-info' : 'badge-neutral'}`}>
                      {link.mode === 'proxy' ? t('links.mode.proxy') : t('links.mode.redirect')}
                    </span>
                  </td>
                  <td className="cell-center">
                    <label className="switch">
                      <input
                        type="checkbox"
                        checked={link.enabled}
                        onChange={() => handleToggleEnable(link)}
                      />
                      <span className="slider"></span>
                    </label>
                  </td>
                  <td className="cell-center">
                    <span className={`badge badge-fixed badge-cache ${link.cache_enabled ? 'badge-success' : 'badge-neutral'}`}>
                      {link.cache_enabled ? t('links.cache.on') : t('links.cache.off')}
                    </span>
                  </td>
                  <td className="action-cell">
                    <div className="table-actions">
                      {link.cache_enabled && (
                        <button
                          className="btn btn-secondary btn-sm table-action-btn table-action-btn-wide"
                          onClick={() => handleClearLinkCache(link.id)}
                          title={t('links.cache.purge_title')}
                        >
                          {t('links.cache.purge')}
                        </button>
                      )}
                      <button
                        className="btn btn-secondary btn-sm table-action-btn"
                        onClick={() => navigate(`/admin/links/${link.id}/edit`)}
                      >
                        {t('links.action.edit')}
                      </button>
                      <button
                        className="btn btn-danger btn-sm table-action-btn"
                        onClick={() => handleDelete(link.id)}
                      >
                        {t('links.action.delete')}
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} style={{ textAlign: 'center', padding: '40px 0', color: 'var(--text-muted)' }}>
                  {t('links.empty')}
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
