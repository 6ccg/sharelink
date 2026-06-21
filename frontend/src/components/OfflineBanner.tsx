import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from '../api/i18n';

/**
 * Global offline detection banner.
 * Displays a sticky top banner when the browser loses network connectivity,
 * and auto-hides when connectivity is restored.
 * Optionally triggers a callback when coming back online so pages can refresh data.
 */
export default function OfflineBanner({ onReconnect }: { onReconnect?: () => void }) {
  const { t } = useTranslation();
  const [isOffline, setIsOffline] = useState(() => !navigator.onLine);

  const handleOnline = useCallback(() => {
    setIsOffline(false);
    onReconnect?.();
  }, [onReconnect]);

  const handleOffline = useCallback(() => {
    setIsOffline(true);
  }, []);

  useEffect(() => {
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [handleOnline, handleOffline]);

  if (!isOffline) return null;

  return (
    <div className="offline-banner">
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
        <line x1="1" y1="1" x2="23" y2="23" />
        <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55" />
        <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39" />
        <path d="M10.71 5.05A16 16 0 0 1 22.56 9" />
        <path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88" />
        <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
        <line x1="12" y1="20" x2="12.01" y2="20" />
      </svg>
      <span>{t('common.offline.banner')}</span>
    </div>
  );
}
