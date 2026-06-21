import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { I18nProvider } from './api/i18n.tsx'
import { ThemeProvider } from './api/theme.tsx'
import { ToastProvider } from './components/Toast.tsx'
import ErrorBoundary from './components/ErrorBoundary.tsx'

// Global unhandledered error handlers — prevent silent white-screen crashes
window.addEventListener('unhandledrejection', (event) => {
  console.error('[Global] Unhandled promise rejection:', event.reason);
});

window.addEventListener('error', (event) => {
  console.error('[Global] Uncaught error:', event.error);
});

createRoot(document.getElementById('root')!, {
  onRecoverableError: (error) => {
    console.warn('[React] Recoverable error:', error);
  },
}).render(
  <StrictMode>
    <ErrorBoundary>
      <I18nProvider>
        <ThemeProvider>
          <ToastProvider>
            <App />
          </ToastProvider>
        </ThemeProvider>
      </I18nProvider>
    </ErrorBoundary>
  </StrictMode>,
)
