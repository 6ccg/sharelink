import React, { useEffect, useState } from 'react';
import { api, getErrorMessage, classifyError } from '../api/client';
import { useTranslation } from '../api/i18n';
import { useToast } from '../components/Toast';
import { TableErrorRow, TableLoadingRow } from '../components/ErrorState';

interface UAPolicy {
  id: number;
  name: string;
  mode: string;
  allow_keywords: string;
  block_keywords: string;
  allow_empty_ua: boolean;
  case_sensitive: boolean;
  match_type: string;
  enabled: boolean;
  created_at: string;
}

export default function UAPolicies() {
  const { t } = useTranslation();
  const toast = useToast();
  const [policies, setPolicies] = useState<UAPolicy[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  // Form State
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showForm, setShowForm] = useState(false);
  
  const [name, setName] = useState('');
  const [mode, setMode] = useState('disabled');
  const [allowText, setAllowText] = useState('');
  const [blockText, setBlockText] = useState('');
  const [allowEmptyUa, setAllowEmptyUa] = useState(true);
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [matchType, setMatchType] = useState('contains');
  const [enabled, setEnabled] = useState(true);

  // Tester State
  const [testPolicyId, setTestPolicyId] = useState('');
  const [testUa, setTestUa] = useState('');
  const [testResult, setTestResult] = useState<{ allowed: boolean; blocked_reason: string } | null>(null);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    fetchPolicies();
  }, []);

  const fetchPolicies = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await api.get('/api/admin/ua-policies');
      setPolicies(data || []);
      if (data && data.length > 0 && !testPolicyId) {
        setTestPolicyId(data[0].id.toString());
      }
    } catch (err) {
      console.error('Failed to load policies:', err);
      setLoadError(classifyError(err, t));
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (p: UAPolicy) => {
    setEditingId(p.id);
    setName(p.name);
    setMode(p.mode);
    setAllowEmptyUa(p.allow_empty_ua);
    setCaseSensitive(p.case_sensitive);
    setMatchType(p.match_type);
    setEnabled(p.enabled);

    // Parse JSON arrays to line-by-line text
    try {
      const allowArr = JSON.parse(p.allow_keywords || '[]');
      setAllowText(allowArr.join('\n'));
    } catch {
      setAllowText('');
    }

    try {
      const blockArr = JSON.parse(p.block_keywords || '[]');
      setBlockText(blockArr.join('\n'));
    } catch {
      setBlockText('');
    }

    setShowForm(true);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const handleDelete = async (id: number) => {
    if (!confirm(t('ua.confirm_delete'))) return;
    try {
      await api.delete(`/api/admin/ua-policies/${id}`);
      toast.success(t('ua.success.deleted'));
      fetchPolicies();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('ua.error.delete_failed')));
    }
  };

  const handleCreateNew = () => {
    setEditingId(null);
    setName('');
    setMode('blacklist');
    setAllowText('');
    setBlockText('');
    setAllowEmptyUa(true);
    setCaseSensitive(false);
    setMatchType('contains');
    setEnabled(true);
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Convert line-by-line keywords to JSON array strings
    const allowArr = allowText.split('\n').map((s) => s.trim()).filter(Boolean);
    const blockArr = blockText.split('\n').map((s) => s.trim()).filter(Boolean);

    const payload = {
      name,
      mode,
      allow_keywords: JSON.stringify(allowArr),
      block_keywords: JSON.stringify(blockArr),
      allow_empty_ua: allowEmptyUa,
      case_sensitive: caseSensitive,
      match_type: matchType,
      enabled,
    };

    try {
      if (editingId) {
        await api.put(`/api/admin/ua-policies/${editingId}`, payload);
        toast.success(t('ua.success.updated'));
      } else {
        await api.post('/api/admin/ua-policies', payload);
        toast.success(t('ua.success.created'));
      }
      setShowForm(false);
      fetchPolicies();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('ua.error.save_failed')));
    }
  };

  const handleTest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testPolicyId) return;
    setTesting(true);
    setTestResult(null);

    try {
      const res = await api.post('/api/admin/ua-policies/test', {
        policy_id: Number(testPolicyId),
        user_agent: testUa,
      });
      setTestResult(res);
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('ua.error.test_failed')));
    } finally {
      setTesting(false);
    }
  };

  return (
    <div className="animate-slideup">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t('ua.title')}</h1>
          <div className="page-subtitle">
            {t('ua.subtitle')}
          </div>
        </div>
        {!showForm && (
          <button className="btn btn-primary" onClick={handleCreateNew}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            {t('ua.create')}
          </button>
        )}
      </div>

      {showForm && (
        <div className="card" style={{ marginBottom: '32px' }}>
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
            {editingId ? t('ua.edit') : t('ua.create')}
          </h3>

          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div className="ua-policy-form-stack">
              <div className="form-group">
                <label className="form-label">{t('ua.field.name')}</label>
                <input
                  type="text"
                  className="form-input"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={t('ua.placeholder.name')}
                  required
                />
              </div>
            </div>

            <div className="ua-policy-form-stack">
              <div className="form-group">
                <label className="form-label">{t('ua.field.operation_mode')}</label>
                <select className="form-select" value={mode} onChange={(e) => setMode(e.target.value)}>
                  <option value="disabled">{t('ua.mode.disabled_full')}</option>
                  <option value="blacklist">{t('ua.mode.blacklist_full')}</option>
                  <option value="whitelist">{t('ua.mode.whitelist_full')}</option>
                  <option value="mixed">{t('ua.mode.mixed_full')}</option>
                </select>
              </div>
            </div>

            <div className="ua-policy-form-stack">
              <div className="form-group">
                <label className="form-label">{t('ua.field.match_type')}</label>
                <select className="form-select" value={matchType} onChange={(e) => setMatchType(e.target.value)}>
                  <option value="contains">{t('ua.field.match.contains')}</option>
                  <option value="regex">{t('ua.field.match.regex')}</option>
                </select>
              </div>
            </div>

            <div className="ua-policy-toggle-row">
                <label className="form-checkbox-label">
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={allowEmptyUa}
                    onChange={(e) => setAllowEmptyUa(e.target.checked)}
                  />
                  {t('ua.col.empty')}
                </label>
                <label className="form-checkbox-label">
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={caseSensitive}
                    onChange={(e) => setCaseSensitive(e.target.checked)}
                  />
                  {t('ua.field.case_sensitive_short')}
                </label>
                <label className="form-checkbox-label">
                  <input
                    type="checkbox"
                    className="form-checkbox"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                  />
                  {t('ua.field.enabled')}
                </label>
            </div>

            <div className="ua-policy-form-stack">
              <div className="form-group">
                <label className="form-label">{t('ua.field.block_kws')}</label>
                <textarea
                  className="form-textarea"
                  value={blockText}
                  onChange={(e) => setBlockText(e.target.value)}
                  placeholder="curl&#10;wget&#10;python-requests&#10;Go-http-client"
                  rows={5}
                  disabled={mode === 'disabled' || mode === 'whitelist'}
                />
              </div>
            </div>

            <div className="ua-policy-form-stack">
              <div className="form-group">
                <label className="form-label">{t('ua.field.allow_kws')}</label>
                <textarea
                  className="form-textarea"
                  value={allowText}
                  onChange={(e) => setAllowText(e.target.value)}
                  placeholder="Mozilla&#10;Chrome&#10;Safari"
                  rows={5}
                  disabled={mode === 'disabled' || mode === 'blacklist'}
                />
              </div>
            </div>

            <div style={{ display: 'flex', gap: '16px', justifyContent: 'flex-end', borderTop: '1px solid var(--border-glass)', paddingTop: '20px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>
                {t('ua.action.cancel')}
              </button>
              <button type="submit" className="btn btn-primary">
                {t('ua.action.save')}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Stack: Policies List and Tester stacked vertically */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '32px' }}>
        {/* Policies List */}
        <div className="card">
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
            {t('ua.list.title')}
          </h3>
          <div className="table-container" style={{ border: 'none' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t('ua.field.name')}</th>
                  <th>{t('ua.col.mode')}</th>
                  <th>{t('ua.col.status')}</th>
                  <th style={{ textAlign: 'right' }}>{t('ua.col.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <TableLoadingRow colSpan={4} label={t('ua.list.loading')} />
                ) : loadError ? (
                  <TableErrorRow colSpan={4} message={loadError} onRetry={fetchPolicies} />
                ) : policies.length > 0 ? (
                  policies.map((p) => (
                    <tr key={p.id}>
                      <td style={{ fontWeight: 600 }}>{p.name}</td>
                      <td>
                        <span className={`badge ${p.mode === 'disabled' ? 'badge-neutral' : p.mode === 'whitelist' ? 'badge-info' : 'badge-warning'}`}>
                          {p.mode === 'disabled' ? t('ua.mode.disabled') : p.mode === 'whitelist' ? t('ua.mode.whitelist') : p.mode === 'blacklist' ? t('ua.mode.blacklist') : t('ua.mode.mixed')}
                        </span>
                      </td>
                      <td>
                        <span className={`badge ${p.enabled ? 'badge-success' : 'badge-neutral'}`}>
                          {p.enabled ? t('ua.status.active') : t('ua.status.inactive')}
                        </span>
                      </td>
                      <td style={{ textAlign: 'right' }}>
                        <div style={{ display: 'inline-flex', gap: '8px' }}>
                          <button className="btn btn-secondary btn-sm" onClick={() => handleEdit(p)}>{t('links.action.edit')}</button>
                          <button className="btn btn-danger btn-sm" onClick={() => handleDelete(p.id)}>{t('links.action.delete')}</button>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={4} style={{ textAlign: 'center', padding: '24px 0', color: 'var(--text-muted)' }}>
                      {t('ua.list.empty')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* User-Agent Tester */}
        <div className="card" style={{ height: 'fit-content' }}>
          <h3 style={{ fontFamily: 'var(--font-title)', fontSize: '18px', fontWeight: 700, marginBottom: '20px' }}>
            {t('ua.tester.title')}
          </h3>
          <form onSubmit={handleTest} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div className="form-group">
              <label className="form-label">{t('ua.tester.policy')}</label>
              <select className="form-select" value={testPolicyId} onChange={(e) => setTestPolicyId(e.target.value)} required>
                <option value="">{t('ua.tester.choose')}</option>
                {policies.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">{t('ua.tester.user_agent')}</label>
              <textarea
                className="form-textarea"
                value={testUa}
                onChange={(e) => setTestUa(e.target.value)}
                placeholder="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
                rows={4}
                required
              />
            </div>

            <button type="submit" className="btn btn-accent" disabled={testing || !testPolicyId}>
              {testing ? t('ua.tester.testing') : t('ua.tester.run')}
            </button>
          </form>

          {testResult && (
            <div style={{ marginTop: '24px', padding: '20px', borderRadius: '8px', border: '1px solid var(--border-glass)', background: 'rgba(255,255,255,0.01)', animation: 'slideUp 0.3s ease' }}>
              <div style={{ fontSize: '14px', color: 'var(--text-secondary)', fontWeight: 600 }}>
                {t('ua.tester.result')}
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginTop: '10px' }}>
                <span className={`badge ${testResult.allowed ? 'badge-success' : 'badge-error'}`} style={{ fontSize: '14px', padding: '6px 16px' }}>
                  {testResult.allowed ? t('ua.tester.allowed') : t('ua.tester.blocked')}
                </span>
                {!testResult.allowed && (
                  <span style={{ fontSize: '14px', color: 'var(--text-secondary)' }}>
                    {t('ua.tester.reason')}: <code>{testResult.blocked_reason}</code>
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
