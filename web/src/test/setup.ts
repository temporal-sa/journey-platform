import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { beforeAll, afterEach, afterAll, expect } from 'vitest';
import { setupServer } from 'msw/node';
import { handlers } from '../mocks/handlers';
import * as axeMatchers from 'vitest-axe/matchers.js';

expect.extend(axeMatchers as any);

if (typeof window !== 'undefined') {
  class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  window.ResizeObserver = window.ResizeObserver || (ResizeObserverMock as unknown as typeof ResizeObserver);
}

export const server = setupServer(...handlers);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());

