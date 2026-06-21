import React, { useEffect, useState } from 'react';
import { api, getErrorMessage, classifyError } from '../api/client';
import { useTranslation } from '../api/i18n';
import { useToast } from '../components/Toast';
import { ErrorState, LoadingState } from '../components/ErrorState';

export default function Settings() {
  const { t } = useTranslation();
  const toast = useToast();
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Password fields
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passLoading, setPassLoading] = useState(false);

  useEffect(() => {
    fetchSettings();
  }, []);

  const fetchSettings = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await api.get('/api/admin/settings');
      setSettings(data || {});
    } catch (err) {
      console.error('Failed to load settings:', err);
      setLoadError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateSetting = (key: string, value: string) => {
    setSettings((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const handleSubmitSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);

    try {
      await api.put('/api/admin/settings', settings);
      toast.success(t('settings.success.save'));
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('settings.error.save_failed')));
    } finally {
      setSaving(false);
    }
  };

  const handleSubmitPassword = async (e: React.FormEvent) => {
    e.preventDefault();

    if (newPassword !== confirmPassword) {
      toast.error(t('settings.error.pass_mismatch'));
      return;
    }

    setPassLoading(true);
    try {
      await api.post('/api/admin/settings/password', {
        old_password: oldPassword,
        new_password: newPassword,
      });
      toast.success(t('settings.success.password'));
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('settings.error.password_failed')));
    } finally {
      setPassLoading(false);
    }
  };

  if (loading) {
    return <LoadingState label={t('settings.loading')} />;
  }

  if (loadError) {
    return (
      <div className="animate-slideup">
        <div className="page-header">
          <div>
            <h1 className="page-title">{t('settings.title')}</h1>
            <div className="page-subtitle">{t('settings.subtitle')}</div>
          </div>
        </div>
        <div className="card">
          <ErrorState message={loadError} onRetry={fetchSettings} />
        </div>
      </div>
    );
  }

  return (
    <div className="animate-slideup" style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('settings.title')}</h1>
          <div className="page-subtitle">
            {t('settings.subtitle')}
          </div>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(360px, 1fr))', gap: '32px' }}>
        {/* Configurations Form */}
        <div className="card">
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '24px' }}>
            {t('settings.section.global')}
          </h3>

          <form onSubmit={handleSubmitSettings} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div className="form-row">
              <div className="form-group">
                <label className="form-label">{t('settings.field.global_cache')}</label>
                <label className="form-checkbox-label" style={{ marginTop: '10px' }}>
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={settings.global_cache_enabled === 'true' || settings.global_cache_enabled === '1'}
                    onChange={(e) => handleUpdateSetting('global_cache_enabled', e.target.checked ? 'true' : 'false')}
                  />
                  {t('settings.field.global_cache_enabled')}
                </label>
              </div>

              <div className="form-group">
                <label className="form-label">{t('settings.field.global_cache_limit')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.global_cache_max_memory_mb || '64'}
                  onChange={(e) => handleUpdateSetting('global_cache_max_memory_mb', e.target.value)}
                  min="1"
                  required
                />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">{t('settings.field.max_proxy_response')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.max_proxy_response_size_mb || '5'}
                  onChange={(e) => handleUpdateSetting('max_proxy_response_size_mb', e.target.value)}
                  min="1"
                  required
                />
                <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>
                  {t('settings.help.max_proxy_response')}
                </span>
              </div>

              <div className="form-group">
                <label className="form-label">{t('settings.field.upstream_connect_timeout')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.upstream_connect_timeout || '10'}
                  onChange={(e) => handleUpdateSetting('upstream_connect_timeout', e.target.value)}
                  min="1"
                  required
                />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label">{t('settings.field.upstream_header_timeout')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.upstream_response_header_timeout || '15'}
                  onChange={(e) => handleUpdateSetting('upstream_response_header_timeout', e.target.value)}
                  min="1"
                  required
                />
              </div>

              <div className="form-group">
                <label className="form-label">{t('settings.field.proxy_total_timeout')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.proxy_total_timeout || '60'}
                  onChange={(e) => handleUpdateSetting('proxy_total_timeout', e.target.value)}
                  min="1"
                  required
                />
              </div>
            </div>

            <div className="form-row" style={{ borderTop: '1px solid var(--border-glass)', paddingTop: '20px' }}>
              <div className="form-group">
                <label className="form-label">{t('settings.field.log_cleanup_label')}</label>
                <label className="form-checkbox-label" style={{ marginTop: '10px' }}>
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={settings.log_cleanup_enabled !== 'false' && settings.log_cleanup_enabled !== '0'}
                    onChange={(e) => handleUpdateSetting('log_cleanup_enabled', e.target.checked ? 'true' : 'false')}
                  />
                  {t('settings.field.log_cleanup')}
                </label>
              </div>

              <div className="form-group">
                <label className="form-label">{t('settings.field.log_retention')}</label>
                <input
                  type="number"
                  className="form-input"
                  value={settings.log_retention_days || '90'}
                  onChange={(e) => handleUpdateSetting('log_retention_days', e.target.value)}
                  min="1"
                  disabled={settings.log_cleanup_enabled === 'false' || settings.log_cleanup_enabled === '0'}
                  required
                />
              </div>
            </div>

            <div className="form-row" style={{ borderTop: '1px solid var(--border-glass)', paddingTop: '20px' }}>
              <div className="form-group">
                <label className="form-label">{t('settings.field.geoip_label')}</label>
                <label className="form-checkbox-label" style={{ marginTop: '10px' }}>
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={settings.geoip_enabled === 'true' || settings.geoip_enabled === '1'}
                    onChange={(e) => handleUpdateSetting('geoip_enabled', e.target.checked ? 'true' : 'false')}
                  />
                  {t('settings.field.geoip')}
                </label>
              </div>

              <div className="form-group">
                <label className="form-label">{t('settings.field.trust_proxy')}</label>
                <label className="form-checkbox-label" style={{ marginTop: '10px' }}>
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={settings.trust_proxy_headers === 'true' || settings.trust_proxy_headers === '1'}
                    onChange={(e) => handleUpdateSetting('trust_proxy_headers', e.target.checked ? 'true' : 'false')}
                  />
                  {t('settings.field.trust_xff')}
                </label>
              </div>
            </div>

            <button type="submit" className="btn btn-primary" style={{ marginTop: '10px' }} disabled={saving}>
              {saving ? t('settings.action.saving') : t('settings.action.save')}
            </button>
          </form>
        </div>

        {/* Change Password Form */}
        <div className="card" style={{ height: 'fit-content' }}>
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '24px' }}>
            {t('settings.section.password')}
          </h3>

          <form onSubmit={handleSubmitPassword} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div className="form-group">
              <label className="form-label">{t('settings.field.curr_pass')}</label>
              <input
                type="password"
                className="form-input"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder={t('settings.placeholder.curr_pass')}
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.field.new_pass')}</label>
              <input
                type="password"
                className="form-input"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder={t('settings.placeholder.new_pass')}
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">{t('settings.field.confirm_pass')}</label>
              <input
                type="password"
                className="form-input"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('settings.placeholder.confirm_pass')}
                required
              />
            </div>

            <button type="submit" className="btn btn-accent" disabled={passLoading}>
              {passLoading ? t('settings.action.updating_pass') : t('settings.action.update_pass')}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
