import { useTranslation } from '../api/i18n';

interface ErrorStateProps {
  /** Custom error message override (already localized). */
  message?: string;
  /** Called when the user clicks the "Retry" button. */
  onRetry?: () => void;
}

/**
 * Full-page or in-card error state display.
 * Used when a data fetch fails and there's nothing meaningful to render.
 */
export function ErrorState({ message, onRetry }: ErrorStateProps) {
  const { t } = useTranslation();
  const displayMsg = message || t('common.error.load_failed');

  return (
    <div className="error-state animate-slideup">
      <div className="error-state-icon">
        <svg
          width="26"
          height="26"
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

      <div className="error-state-title">{t('common.error.load_failed')}</div>
      <div className="error-state-message">{displayMsg}</div>

      {onRetry && (
        <button className="btn btn-primary" onClick={onRetry}>
          <svg
            width="14"
            height="14"
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
          {t('common.error.retry')}
        </button>
      )}
    </div>
  );
}

interface TableErrorRowProps {
  colSpan: number;
  message?: string;
  onRetry?: () => void;
}

/**
 * Error row rendered inside a <table> when the data fetch fails.
 */
export function TableErrorRow({ colSpan, message, onRetry }: TableErrorRowProps) {
  const { t } = useTranslation();
  const displayMsg = message || t('common.error.load_failed');

  return (
    <tr className="table-state-row">
      <td colSpan={colSpan}>
        <div className="table-error-cell">
          <svg
            width="22"
            height="22"
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
          <span className="error-msg">{displayMsg}</span>
          {onRetry && (
            <button className="btn btn-secondary btn-sm" onClick={onRetry}>
              <svg
                width="12"
                height="12"
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
              {t('common.error.retry')}
            </button>
          )}
        </div>
      </td>
    </tr>
  );
}

/**
 * Animated loading dots used in place of plain text "loading...".
 */
export function LoadingState({ label }: { label?: string }) {
  return (
    <div className="loading-state">
      <div className="loading-dot-row">
        <span className="loading-dot" />
        <span className="loading-dot" />
        <span className="loading-dot" />
      </div>
      {label && <span className="loading-text">{label}</span>}
    </div>
  );
}

/**
 * Inline loading row for tables.
 */
export function TableLoadingRow({ colSpan, label }: { colSpan: number; label?: string }) {
  return (
    <tr className="table-state-row">
      <td colSpan={colSpan} style={{ color: 'var(--text-secondary)' }}>
        <div className="table-error-cell" style={{ color: 'inherit' }}>
          <div className="loading-dot-row">
            <span className="loading-dot" />
            <span className="loading-dot" />
            <span className="loading-dot" />
          </div>
          {label && <span style={{ fontSize: '14px' }}>{label}</span>}
        </div>
      </td>
    </tr>
  );
}
