import { useEffect, useState } from 'react';

function getMatches(query: string): boolean {
  if (typeof window !== 'undefined') {
    return window.matchMedia(query).matches;
  }
  return false;
}

export function useMediaQuery(query: string) {
  const [value, setValue] = useState(() => getMatches(query));

  useEffect(() => {
    const result = matchMedia(query);

    function onChange(event: MediaQueryListEvent) {
      setValue(event.matches);
    }

    result.addEventListener('change', onChange);
    return () => result.removeEventListener('change', onChange);
  }, [query]);

  return value;
}
