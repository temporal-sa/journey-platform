import { useState, useEffect, useCallback, useRef } from 'react';

export function useRouteParams<T extends Record<string, string>>(defaults: T) {
  const defaultsRef = useRef(defaults);

  useEffect(() => {
    defaultsRef.current = defaults;
  }, [JSON.stringify(defaults)]);

  const getSearchParams = useCallback((): T => {
    if (typeof window === 'undefined') return defaultsRef.current;
    const rawHash = window.location.hash.replace(/^#\/?/, '');
    const [, queryPart] = rawHash.split('?');
    const search = new URLSearchParams(queryPart !== undefined ? queryPart : window.location.search);
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
      if (typeof window !== 'undefined') {
        const rawHash = window.location.hash.replace(/^#\/?/, '');
        const [pathPart, queryPart] = rawHash.split('?');
        const search = new URLSearchParams(queryPart || '');

        Object.entries(newParams).forEach(([k, v]) => {
          if (v !== undefined && v !== '') {
            search.set(k, v as string);
          } else {
            search.delete(k);
          }
        });

        const newQuery = search.toString();
        const newHash = `#/${pathPart}${newQuery ? '?' + newQuery : ''}`;
        if (window.location.hash !== newHash) {
          window.history.replaceState(null, '', newHash);
        }
      }
      return next;
    });
  }, []);

  return [params, setParams] as const;
}
