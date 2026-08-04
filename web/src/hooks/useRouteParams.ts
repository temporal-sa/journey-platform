import { useState, useEffect, useCallback, useRef } from 'react';

export function useRouteParams<T extends Record<string, string>>(defaults: T) {
  const defaultsRef = useRef(defaults);

  useEffect(() => {
    defaultsRef.current = defaults;
  }, [JSON.stringify(defaults)]);

  const getSearchParams = useCallback((): T => {
    if (typeof window === 'undefined') return defaultsRef.current;
    const search = new URLSearchParams(window.location.search);
    const result = { ...defaultsRef.current };
    for (const key of Object.keys(defaultsRef.current)) {
      const val = search.get(key);
      if (val !== null) {
        (result as Record<string, string>)[key] = val;
      }
    }
    return result;
  }, []);

  const [params, setParamsState] = useState<T>(getSearchParams);

  useEffect(() => {
    const handlePopState = () => {
      setParamsState(getSearchParams());
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, [getSearchParams]);

  const setParams = useCallback((newParams: Partial<T>) => {
    setParamsState((prev) => {
      let hasChange = false;
      const next = { ...prev };
      for (const [k, v] of Object.entries(newParams)) {
        if (prev[k] !== v) {
          hasChange = true;
          if (v !== undefined) {
            (next as any)[k] = v;
          }
        }
      }
      if (!hasChange) return prev;

      if (typeof window !== 'undefined') {
        const url = new URL(window.location.href);
        Object.entries(next).forEach(([k, v]) => {
          if (v !== undefined && v !== '') {
            url.searchParams.set(k, v as string);
          } else {
            url.searchParams.delete(k);
          }
        });
        window.history.pushState({}, '', url.toString());
      }
      return next;
    });
  }, []);

  return [params, setParams] as const;
}
