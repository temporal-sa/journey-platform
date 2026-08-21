// Simulated API Failure Mode Helper & State

let isFailureEnabledState = false;
let isActivityFailureEnabledState = false;
let simulatedActivityLatencyMSState = 0;

const STORAGE_KEY = 'agy_simulated_api_failure_enabled';
const ACTIVITY_STORAGE_KEY = 'agy_simulated_activity_failure_enabled';
const LATENCY_STORAGE_KEY = 'agy_simulated_activity_latency_ms';
if (typeof window !== 'undefined' && window.localStorage) {
  try {
    isFailureEnabledState = localStorage.getItem(STORAGE_KEY) === 'true';
    isActivityFailureEnabledState = localStorage.getItem(ACTIVITY_STORAGE_KEY) === 'true';
    const storedLatency = localStorage.getItem(LATENCY_STORAGE_KEY);
    if (storedLatency) {
      const parsed = parseInt(storedLatency, 10);
      if (!isNaN(parsed) && parsed >= 0) {
        simulatedActivityLatencyMSState = parsed;
      }
    }
  } catch {
    // Ignore localStorage access errors
  }
}

/**
 * Helper method to detect whether simulated API failure mode is currently enabled throughout the application.
 */
export function isSimulatedApiFailureEnabled(): boolean {
  return isFailureEnabledState;
}

/**
 * Sets the simulated API failure mode state.
 */
export function setSimulatedApiFailureEnabled(enabled: boolean): void {
  isFailureEnabledState = enabled;
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.setItem(STORAGE_KEY, enabled ? 'true' : 'false');
    } catch {
      // Ignore
    }
  }
}

/**
 * Toggles the simulated API failure mode state and returns the new state.
 */
export function toggleSimulatedApiFailure(): boolean {
  const nextState = !isFailureEnabledState;
  setSimulatedApiFailureEnabled(nextState);
  return nextState;
}

/**
 * Helper method to detect whether simulated Activity failure mode is currently enabled.
 */
export function isSimulatedActivityFailureEnabled(): boolean {
  return isActivityFailureEnabledState;
}

/**
 * Sets the simulated Activity failure mode state.
 */
export function setSimulatedActivityFailureEnabled(enabled: boolean): void {
  isActivityFailureEnabledState = enabled;
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.setItem(ACTIVITY_STORAGE_KEY, enabled ? 'true' : 'false');
    } catch {
      // Ignore
    }
  }
}

/**
 * Toggles the simulated Activity failure mode state and returns the new state.
 */
export function toggleSimulatedActivityFailure(): boolean {
  const nextState = !isActivityFailureEnabledState;
  setSimulatedActivityFailureEnabled(nextState);
  return nextState;
}
/**
 * Gets the simulated Activity latency in milliseconds.
 */
export function getSimulatedActivityLatencyMS(): number {
  return simulatedActivityLatencyMSState;
}

/**
 * Sets the simulated Activity latency in milliseconds.
 */
export function setSimulatedActivityLatencyMS(latencyMs: number): void {
  const clamped = Math.max(0, latencyMs || 0);
  simulatedActivityLatencyMSState = clamped;
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.setItem(LATENCY_STORAGE_KEY, String(clamped));
    } catch {
      // Ignore
    }
  }
}

