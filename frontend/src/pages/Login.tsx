import React, { useState, useEffect } from 'react';
import { useLocation } from '../api/router';
import { api, getErrorMessage } from '../api/client';
import { useTranslation } from '../api/i18n';
import { useTheme } from '../api/theme';
import { useToast } from '../components/Toast';

export default function Login() {
  const { t, lang, setLang } = useTranslation();
  const { theme, toggleTheme } = useTheme();
  const toast = useToast();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [, navigate] = useLocation();

  // Show a one-time hint if user was redirected here due to auth failure
  useEffect(() => {
    const hint = sessionStorage.getItem('login_hint');
    if (hint) {
      toast.warning(hint);
      sessionStorage.removeItem('login_hint');
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      const data = await api.post('/api/auth/login', { username, password });
      localStorage.setItem('sharelink_token', data.token);
      navigate('/admin');
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('login.failed')));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      {/* Floating Language & Theme Switcher */}
      <div className="lang-switcher-floating" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <button 
          type="button"
          className="lang-switcher-btn"
          onClick={toggleTheme}
          style={{ display: 'flex', alignItems: 'center', padding: '6px', cursor: 'pointer' }}
          title={theme === 'dark' ? t('common.theme.light') : t('common.theme.dark')}
        >
          {theme === 'dark' ? (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="5"></circle>
              <line x1="12" y1="1" x2="12" y2="3"></line>
              <line x1="12" y1="21" x2="12" y2="23"></line>
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
              <line x1="1" y1="12" x2="3" y2="12"></line>
              <line x1="21" y1="12" x2="23" y2="12"></line>
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
            </svg>
          ) : (
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          )}
        </button>
        <div style={{ width: '1px', height: '14px', background: 'var(--border-glass)' }}></div>
        <button 
          type="button"
          className={`lang-switcher-btn ${lang === 'zh' ? 'active' : ''}`}
          onClick={() => setLang('zh')}
        >
          {t('common.language.zh')}
        </button>
        <button 
          type="button"
          className={`lang-switcher-btn ${lang === 'en' ? 'active' : ''}`}
          onClick={() => setLang('en')}
        >
          {t('common.language.en_full')}
        </button>
      </div>

      <div className="login-card animate-slideup">
        <div className="login-logo">{t('login.title')}</div>
        <div className="login-subtitle">{t('login.subtitle')}</div>

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label" htmlFor="username">{t('login.username')}</label>
            <input
              type="text"
              id="username"
              className="form-input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </div>

          <div className="form-group" style={{ marginBottom: '32px' }}>
            <label className="form-label" htmlFor="password">{t('login.password')}</label>
            <input
              type="password"
              id="password"
              className="form-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
            />
          </div>

          <button
            type="submit"
            className="btn btn-primary"
            style={{ width: '100%', padding: '12px' }}
            disabled={loading}
          >
            {loading ? t('login.loggingin') : t('login.signin')}
          </button>
        </form>
      </div>
    </div>
  );
}
