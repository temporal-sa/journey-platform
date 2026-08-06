import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  isSimulatedApiFailureEnabled,
  setSimulatedApiFailureEnabled,
  toggleSimulatedApiFailure,
} from './simulatedFailure';
import { JourneyApiClient, APIError } from './client';

describe('Simulated API Failure Helper & Integration', () => {
  beforeEach(() => {
    setSimulatedApiFailureEnabled(false);
  });

  afterEach(() => {
    setSimulatedApiFailureEnabled(false);
  });

  it('defaults to disabled state', () => {
    expect(isSimulatedApiFailureEnabled()).toBe(false);
  });

  it('enables and disables simulated failure state correctly', () => {
    setSimulatedApiFailureEnabled(true);
    expect(isSimulatedApiFailureEnabled()).toBe(true);

    setSimulatedApiFailureEnabled(false);
    expect(isSimulatedApiFailureEnabled()).toBe(false);
  });

  it('toggles simulated failure state correctly', () => {
    const state1 = toggleSimulatedApiFailure();
    expect(state1).toBe(true);
    expect(isSimulatedApiFailureEnabled()).toBe(true);

    const state2 = toggleSimulatedApiFailure();
    expect(state2).toBe(false);
    expect(isSimulatedApiFailureEnabled()).toBe(false);
  });

  it('causes JourneyApiClient requests to throw APIError (503) when failure mode is enabled', async () => {
    const client = new JourneyApiClient();
    setSimulatedApiFailureEnabled(true);

    await expect(
      client.getJourneyDraft('draft-101', { 'X-Tenant-ID': 'default' })
    ).rejects.toThrow(APIError);

    try {
      await client.getJourneyDraft('draft-101', { 'X-Tenant-ID': 'default' });
    } catch (err) {
      expect(err).toBeInstanceOf(APIError);
      const apiErr = err as APIError;
      expect(apiErr.status).toBe(503);
      expect(apiErr.code).toBe('SIMULATED_API_FAILURE');
      expect(apiErr.message).toContain('[DEV SIMULATION] Service Unavailable');
    }
  });

  it('causes publishJourneyDraft to throw SIMULATED_API_FAILURE when failure mode is enabled', async () => {
    const client = new JourneyApiClient();
    setSimulatedApiFailureEnabled(true);

    await expect(
      client.publishJourneyDraft('draft-101', { 'X-Tenant-ID': 'default' })
    ).rejects.toThrow(APIError);
  });
});
