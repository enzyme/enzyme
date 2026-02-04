/* eslint-disable react-refresh/only-export-components */
import type { ReactElement, ReactNode } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, type MemoryRouterProps } from 'react-router-dom';

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

interface ProvidersProps {
  children: ReactNode;
  queryClient?: QueryClient;
  routerProps?: MemoryRouterProps;
}

function Providers({ children, queryClient, routerProps }: ProvidersProps) {
  const client = queryClient ?? createTestQueryClient();

  return (
    <QueryClientProvider client={client}>
      <MemoryRouter {...routerProps}>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient;
  routerProps?: MemoryRouterProps;
}

function customRender(ui: ReactElement, options: CustomRenderOptions = {}) {
  const { queryClient, routerProps, ...renderOptions } = options;

  return render(ui, {
    wrapper: ({ children }) => (
      <Providers queryClient={queryClient} routerProps={routerProps}>
        {children}
      </Providers>
    ),
    ...renderOptions,
  });
}

export { customRender as render };
