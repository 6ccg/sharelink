import { useEffect, useState, useCallback } from 'react';
import { useLocation, matchPath } from './api/router';
import { api, classifyError } from './api/client';
import { useTranslation } from './api/i18n';
import Sidebar from './components/Sidebar';
import OfflineBanner from './components/OfflineBanner';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Links from './pages/Links';
import LinkEdit from './pages/LinkEdit';
import UAPolicies from './pages/UAPolicies';
import Cache from './pages/Cache';
import Logs from './pages/Logs';
import Settings from './pages/Settings';

export default function App() {
  const { t } = useTranslation();
  const [path, navigate] = useLocation();
  const [loading, setLoading] = useState(true);
  const [authenticated, setAuthenticated] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const [sidebarOpen, setSidebarOpen] = useState(() => {
    if (window.innerWidth <= 768) return false;
    const saved = localStorage.getItem('sharelink_sidebar_expanded');
    return saved !== null ? saved === 'true' : true;
  });

  const toggleSidebar = useCallback((open: boolean) => {
    setSidebarOpen(open);
    if (window.innerWidth > 768) {
      localStorage.setItem('sharelink_sidebar_expanded', String(open));
    }
  }, []);

  useEffect(() => {
    const handleResize = () => {
      if (window.innerWidth <= 768) {
        setSidebarOpen(false);
      } else {
        const saved = localStorage.getItem('sharelink_sidebar_expanded');
        setSidebarOpen(saved !== null ? saved === 'true' : true);
      }
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Re-render all pages when coming back online
  const handleReconnect = useCallback(() => {
    setRefreshKey((k) => k + 1);
  }, []);

  useEffect(() => {
    checkAuth();
  }, [path]);

  const checkAuth = async () => {
    const token = localStorage.getItem('sharelink_token');

    // If path is root '/', redirect to '/admin' or '/login'
    if (path === '/') {
      if (token) {
        navigate('/admin');
      } else {
        navigate('/login');
      }
      return;
    }

    if (!token) {
      setAuthenticated(false);
      setLoading(false);
      if (path.startsWith('/admin')) {
        navigate('/login');
      }
      return;
    }

    // Verify token validity
    try {
      await api.get('/api/admin/auth/me');
      setAuthenticated(true);
      if (path === '/login') {
        navigate('/admin');
      }
    } catch (err) {
      localStorage.removeItem('sharelink_token');
      setAuthenticated(false);
      // Show a meaningful reason when redirected to login
      const reason = classifyError(err, t);
      sessionStorage.setItem('login_hint', reason);
      navigate('/login');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    if (path.startsWith('/admin') && localStorage.getItem('sharelink_token')) {
      return (
        <div className="app-container">
          <OfflineBanner />
          <Sidebar isOpen={sidebarOpen} onClose={() => toggleSidebar(false)} />
          <main className="main-content">
            <div className="auth-check-panel">
              <span className="auth-check-dot" />
              <span>{t('common.auth.verifying')}</span>
            </div>
          </main>
        </div>
      );
    }

    return (
      <div className="auth-check-screen">
        <div className="auth-check-panel">
          <span className="auth-check-dot" />
          <span>{t('common.auth.verifying')}</span>
        </div>
      </div>
    );
  }

  // 1. Render Login Screen
  if (path === '/login' || !authenticated) {
    return <Login />;
  }

  const renderPage = () => {
    if (path === '/admin') {
      return <Dashboard />;
    }
    if (path === '/admin/links') {
      return <Links />;
    }
    if (path === '/admin/links/new') {
      return <LinkEdit />;
    }
    if (matchPath('/admin/links/:id/edit', path)) {
      return <LinkEdit />;
    }
    if (path === '/admin/ua-policies') {
      return <UAPolicies />;
    }
    if (path === '/admin/cache') {
      return <Cache />;
    }
    if (path === '/admin/logs') {
      return <Logs />;
    }
    if (path === '/admin/settings') {
      return <Settings />;
    }
    return (
      <div className="card" style={{ textAlign: 'center', marginTop: '40px' }}>
        <div className="page-404">
          <div className="page-404-code">404</div>
          <div className="page-404-title">{t('common.404.title')}</div>
          <div className="page-404-body">{t('common.404.body')}</div>
          <button className="btn btn-primary" onClick={() => navigate('/admin')}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
              <polyline points="9 22 9 12 15 12 15 22" />
            </svg>
            {t('common.404.back')}
          </button>
        </div>
      </div>
    );
  };

  // 3. Render Admin Panel Layout with Sidebar
  return (
    <div className="app-container">
      <OfflineBanner onReconnect={handleReconnect} />

      {/* Sidebar Toggle Button */}
      <button 
        className="sidebar-toggle" 
        onClick={() => toggleSidebar(!sidebarOpen)}
        style={{ left: sidebarOpen ? 'calc(var(--sidebar-width) + 20px)' : '20px' }}
        aria-label="Toggle Sidebar"
      >
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
          {sidebarOpen ? (
            <line x1="18" y1="6" x2="6" y2="18"></line>
          ) : (
            <>
              <line x1="3" y1="12" x2="21" y2="12"></line>
              <line x1="3" y1="6" x2="21" y2="6"></line>
              <line x1="3" y1="18" x2="21" y2="18"></line>
            </>
          )}
        </svg>
      </button>

      {/* Overlay to close sidebar on click (CSS controls display on mobile/desktop) */}
      {sidebarOpen && (
        <div className="sidebar-overlay" onClick={() => toggleSidebar(false)} />
      )}

      <Sidebar isOpen={sidebarOpen} onClose={() => toggleSidebar(false)} />
      <main className={`main-content ${sidebarOpen ? 'sidebar-open' : 'sidebar-collapsed'}`} key={refreshKey}>
        {renderPage()}
      </main>
    </div>
  );
}
