import React, { useEffect, useState } from 'react';
import { api, getErrorMessage } from '../api/client';
import { useLocation, matchPath } from '../api/router';
import { useTranslation } from '../api/i18n';
import { useToast } from '../components/Toast';

interface UAPolicy {
  id: number;
  name: string;
  enabled: boolean;
}

export default function LinkEdit() {
  const { t } = useTranslation();
  const toast = useToast();
  const [path, navigate] = useLocation();
  const params = matchPath('/admin/links/:id/edit', path);
  const isEdit = !!params;

  const [prefix, setPrefix] = useState('/export');
  const [slug, setSlug] = useState('');
  const [targetUrl, setTargetUrl] = useState('');
  const [mode, setMode] = useState('proxy');
  const [enabled, setEnabled] = useState(true);
  const [startTime, setStartTime] = useState('');
  const [expireTime, setExpireTime] = useState('');
  
  const [cacheEnabled, setCacheEnabled] = useState(false);
  const [cacheTtl, setCacheTtl] = useState(600);
  const [cacheMaxSize, setCacheMaxSize] = useState(5);
  
  const [filenameMode, setFilenameMode] = useState('inherit');
  const [customFilename, setCustomFilename] = useState('');
  
  const [uaPolicyId, setUaPolicyId] = useState<string>('');
  const [note, setNote] = useState('');

  const [policies, setPolicies] = useState<UAPolicy[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    // Fetch User-Agent Policies
    async function loadPolicies() {
      try {
        const list = await api.get('/api/admin/ua-policies');
        setPolicies(list || []);
      } catch (err) {
        console.error('Failed to load UA policies:', err);
        toast.warning(getErrorMessage(err, t('common.error.load_failed')));
      }
    }
    loadPolicies();

    // If edit mode, load link details
    if (isEdit && params) {
      loadLinkDetails(params.id);
    } else {
      // Create mode: generate a random slug initially
      generateRandomSlug();
    }
  }, [isEdit]);

  const loadLinkDetails = async (id: string) => {
    setLoading(true);
    try {
      const data = await api.get(`/api/admin/links/${id}`);
      setPrefix(data.prefix);
      setSlug(data.slug);
      setTargetUrl(data.target_url);
      setMode(data.mode);
      setEnabled(data.enabled);
      setStartTime(data.start_time ? formatDatetimeLocal(data.start_time) : '');
      setExpireTime(data.expire_time ? formatDatetimeLocal(data.expire_time) : '');
      setCacheEnabled(data.cache_enabled);
      setCacheTtl(data.cache_ttl);
      setCacheMaxSize(data.cache_max_object_size_mb);
      setFilenameMode(data.filename_mode);
      setCustomFilename(data.custom_filename || '');
      setUaPolicyId(data.ua_policy_id ? data.ua_policy_id.toString() : '');
      setNote(data.note || '');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('edit.error.load_failed')));
    } finally {
      setLoading(false);
    }
  };

  const formatDatetimeLocal = (isoString: string) => {
    // Convert ISO string (e.g. 2026-06-20T21:19:06Z) to YYYY-MM-DDTHH:MM local format
    const d = new Date(isoString);
    const pad = (n: number) => (n < 10 ? '0' + n : n);
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  };

  const generateRandomSlug = async () => {
    try {
      const res = await api.post('/api/admin/links/generate-slug');
      setSlug(res.slug);
    } catch {
      setSlug(generateClientSlug());
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    const payload = {
      prefix,
      slug,
      target_url: targetUrl,
      mode,
      enabled,
      start_time: startTime ? new Date(startTime).toISOString() : null,
      expire_time: expireTime ? new Date(expireTime).toISOString() : null,
      cache_enabled: mode === 'proxy' && cacheEnabled,
      cache_ttl: Number(cacheTtl),
      cache_max_object_size_mb: Number(cacheMaxSize),
      filename_mode: mode === 'proxy' ? filenameMode : 'inherit',
      custom_filename: mode === 'proxy' && filenameMode === 'custom' ? customFilename : null,
      ua_policy_id: uaPolicyId ? Number(uaPolicyId) : null,
      note: note || null,
    };

    try {
      if (isEdit && params) {
        await api.put(`/api/admin/links/${params.id}`, payload);
        toast.success(t('edit.success.updated'));
      } else {
        await api.post('/api/admin/links', payload);
        toast.success(t('edit.success.created'));
      }
      navigate('/admin/links');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('edit.error.save_failed')));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="animate-slideup">
      <div className="page-header" style={{ marginBottom: '24px' }}>
        <div>
          <h1 className="page-title">{isEdit ? t('edit.title.edit') : t('edit.title.create')}</h1>
          <div className="page-subtitle">
            {t('edit.subtitle')}
          </div>
        </div>
        <button className="btn btn-secondary" onClick={() => navigate('/admin/links')}>
          {t('edit.action.cancel')}
        </button>
      </div>

      <form onSubmit={handleSubmit} className="card" style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
        <div className="public-path-preview">
          <div className="public-path-preview-label">
            {t('edit.preview')}
          </div>
          <div className="public-path-preview-url">
            {window.location.origin}
            <span className="public-path-preview-prefix">{prefix || '/'}</span>
            <span className="public-path-preview-slug">{slug ? `/${slug}` : ''}</span>
          </div>
        </div>

        <div className="form-row">
          <div className="form-group">
            <label className="form-label">{t('edit.field.prefix')}</label>
            <input
              type="text"
              className="form-input"
              value={prefix}
              onChange={(e) => setPrefix(e.target.value)}
              placeholder="/go"
              disabled={isEdit}
              required
            />
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
              {t('edit.help.prefix')}
            </span>
          </div>

          <div className="form-group">
            <label className="form-label">{t('edit.field.slug')}</label>
            <div className="slug-input-row">
              <input
                type="text"
                className="form-input"
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                placeholder="aZ8k2Lm9Qp"
                disabled={isEdit}
                required
              />
              {!isEdit && (
                <button type="button" className="btn btn-secondary slug-random-btn" onClick={generateRandomSlug}>
                  {t('edit.field.random')}
                </button>
              )}
            </div>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
              {t('edit.help.slug')}
            </span>
          </div>
        </div>

        <div className="form-group">
          <label className="form-label">{t('edit.field.target')}</label>
          <input
            type="text"
            className="form-input"
            value={targetUrl}
            onChange={(e) => setTargetUrl(e.target.value)}
            placeholder={t('edit.field.target.placeholder')}
            required
          />
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
            {t('edit.help.target')}
          </span>
        </div>

        <div className="form-row">
          <div className="form-group">
            <label className="form-label">{t('edit.field.mode')}</label>
            <select className="form-select" value={mode} onChange={(e) => setMode(e.target.value)}>
              <option value="proxy">
                {t('edit.field.mode.proxy_short')}
              </option>
              <option value="redirect">
                {t('edit.field.mode.redirect_short')}
              </option>
            </select>
          </div>

          <div className="form-group">
            <label className="form-label" style={{ marginBottom: '14px' }}>{t('edit.field.status')}</label>
            <label className="form-checkbox-label">
              <input
                type="checkbox"
                className="form-checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              {t('edit.field.enabled')}
            </label>
          </div>
        </div>

        <div className="form-row">
          <div className="form-group">
            <label className="form-label">{t('edit.field.start_time')}</label>
            <input
              type="datetime-local"
              className="form-input"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
            />
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
              {t('edit.help.start_time')}
            </span>
          </div>

          <div className="form-group">
            <label className="form-label">{t('edit.field.expire_time')}</label>
            <input
              type="datetime-local"
              className="form-input"
              value={expireTime}
              onChange={(e) => setExpireTime(e.target.value)}
            />
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
              {t('edit.help.expire_time')}
            </span>
          </div>
        </div>

        {/* Proxy Caching Options */}
        {mode === 'proxy' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', padding: '24px', background: 'rgba(255,255,255,0.01)', border: '1px solid var(--border-glass)', borderRadius: '8px' }}>
            <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '16px', fontWeight: 700, borderBottom: '1px solid var(--border-glass)', paddingBottom: '12px' }}>
              {t('edit.section.proxy_cache')}
            </h3>

            <div className="form-group">
              <label className="form-checkbox-label">
                <input
                  type="checkbox"
                  className="form-checkbox"
                  checked={cacheEnabled}
                  onChange={(e) => setCacheEnabled(e.target.checked)}
                />
                {t('edit.field.cache_link')}
              </label>
            </div>

            {cacheEnabled && (
              <div className="form-row">
                <div className="form-group">
                  <label className="form-label">{t('edit.field.cache_ttl')}</label>
                  <input
                    type="number"
                    className="form-input"
                    value={cacheTtl}
                    onChange={(e) => setCacheTtl(Number(e.target.value))}
                    min="1"
                    required
                  />
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                    {t('edit.help.cache_ttl')}
                  </span>
                </div>

                <div className="form-group">
                  <label className="form-label">{t('edit.field.cache_size')}</label>
                  <input
                    type="number"
                    className="form-input"
                    value={cacheMaxSize}
                    onChange={(e) => setCacheMaxSize(Number(e.target.value))}
                    min="1"
                    required
                  />
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                    {t('edit.help.cache_size')}
                  </span>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Proxy Filename Options */}
        {mode === 'proxy' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px', padding: '24px', background: 'rgba(255,255,255,0.01)', border: '1px solid var(--border-glass)', borderRadius: '8px' }}>
            <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '16px', fontWeight: 700, borderBottom: '1px solid var(--border-glass)', paddingBottom: '12px' }}>
              {t('edit.section.filename')}
            </h3>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">{t('edit.field.filename')}</label>
                <select className="form-select" value={filenameMode} onChange={(e) => setFilenameMode(e.target.value)}>
                  <option value="inherit">{t('edit.field.filename.inherit')}</option>
                  <option value="auto">{t('edit.field.filename.auto')}</option>
                  <option value="custom">{t('edit.field.filename.custom')}</option>
                </select>
              </div>

              {filenameMode === 'custom' && (
                <div className="form-group">
                  <label className="form-label">{t('edit.field.filename.custom_val')}</label>
                  <input
                    type="text"
                    className="form-input"
                    value={customFilename}
                    onChange={(e) => setCustomFilename(e.target.value)}
                    placeholder="my-downloaded-file.zip"
                    required
                  />
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                    {t('edit.help.filename.custom')}
                  </span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Policies and notes */}
        <div className="form-row">
          <div className="form-group">
            <label className="form-label">{t('edit.field.ua_policy')}</label>
            <select className="form-select" value={uaPolicyId} onChange={(e) => setUaPolicyId(e.target.value)}>
              <option value="">{t('edit.field.ua_policy.global')}</option>
              {policies.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name} {!p.enabled && t('edit.status.disabled_option')}
                </option>
              ))}
            </select>
          </div>

          <div className="form-group">
            <label className="form-label">{t('edit.field.admin_note')}</label>
            <textarea
              className="form-textarea"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={t('edit.field.note.placeholder')}
              rows={3}
            />
          </div>
        </div>

        <div style={{ display: 'flex', gap: '16px', justifyContent: 'flex-end', borderTop: '1px solid var(--border-glass)', paddingTop: '20px' }}>
          <button type="button" className="btn btn-secondary" onClick={() => navigate('/admin/links')}>
            {t('edit.action.cancel')}
          </button>
          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? t('edit.action.saving') : t('edit.action.save')}
          </button>
        </div>
      </form>
    </div>
  );
}

function generateClientSlug() {
  const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  const bytes = new Uint8Array(10);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => charset[byte % charset.length]).join('');
}
