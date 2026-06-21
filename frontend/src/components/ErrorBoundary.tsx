import React from 'react';
import { useTranslation } from '../api/i18n';

interface Props {
  children: React.ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('[ErrorBoundary] Uncaught error:', error, info.componentStack);
  }

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      return (
        <ErrorBoundaryFallback
          error={this.state.error}
          onReload={this.handleReload}
        />
      );
    }
    return this.props.children;
  }
}

function ErrorBoundaryFallback({
  error,
  onReload,
}: {
  error: Error | null;
  onReload: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="error-boundary-page">
      <div className="error-boundary-icon">
        <svg
          width="32"
          height="32"
          viewBox="0 0 24 24"
          fill="none"
          stroke="#f87171"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      </div>

      <div className="error-boundary-title">{t('common.error.crash_title')}</div>

      <div className="error-boundary-body">{t('common.error.crash_body')}</div>

      {error?.message && (
        <pre className="error-boundary-details">{error.message}</pre>
      )}

      <button className="btn btn-primary" onClick={onReload}>
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <polyline points="23 4 23 10 17 10" />
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
        </svg>
        {t('common.error.crash_reload')}
      </button>
    </div>
  );
}
