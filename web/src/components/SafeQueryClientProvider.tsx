import React from 'react';
import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';

const defaultQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
  },
});

export function SafeQueryClientProvider({ children }: { children: React.ReactNode }) {
  try {
    const client = useQueryClient();
    if (client) {
      return <>{children}</>;
    }
  } catch {
    // Fallback for isolated unit tests rendered outside top-level QueryClientProvider
  }
  return <QueryClientProvider client={defaultQueryClient}>{children}</QueryClientProvider>;
}
