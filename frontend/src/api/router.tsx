import React, { useState, useEffect } from 'react';

// Custom event key for history pushes
const PUSH_STATE_EVENT = 'sharelink_pushstate';

export function useLocation() {
  const [path, setPath] = useState(window.location.pathname);

  useEffect(() => {
    const handleLocationChange = () => {
      setPath(window.location.pathname);
    };

    window.addEventListener('popstate', handleLocationChange);
    window.addEventListener(PUSH_STATE_EVENT, handleLocationChange);

    return () => {
      window.removeEventListener('popstate', handleLocationChange);
      window.removeEventListener(PUSH_STATE_EVENT, handleLocationChange);
    };
  }, []);

  const navigate = (to: string) => {
    window.history.pushState({}, '', to);
    window.dispatchEvent(new Event(PUSH_STATE_EVENT));
  };

  return [path, navigate] as const;
}

export function matchPath(pattern: string, path: string) {
  const patternParts = pattern.split('/');
  const pathParts = path.split('/');

  if (patternParts.length !== pathParts.length) return null;

  const params: Record<string, string> = {};

  for (let i = 0; i < patternParts.length; i++) {
    if (patternParts[i].startsWith(':')) {
      const paramName = patternParts[i].slice(1);
      params[paramName] = pathParts[i];
    } else if (patternParts[i] !== pathParts[i]) {
      return null;
    }
  }

  return params;
}

interface LinkProps {
  to: string;
  children: React.ReactNode;
  className?: string;
  activeClassName?: string;
  onClick?: (e: React.MouseEvent) => void;
}

export function Link({ to, children, className = '', activeClassName = '', onClick }: LinkProps) {
  const [path, navigate] = useLocation();
  const isActive = path === to || (to !== '/admin' && to !== '/login' && path.startsWith(to));

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    if (onClick) {
      onClick(e);
    }
    navigate(to);
  };

  return (
    <a href={to} onClick={handleClick} className={`${className} ${isActive ? activeClassName : ''}`}>
      {children}
    </a>
  );
}
