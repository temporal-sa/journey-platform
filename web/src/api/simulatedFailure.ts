// Simulated API Failure Mode Helper & State

let isFailureEnabledState = false;

const STORAGE_KEY = 'agy_simulated_api_failure_enabled';

if (typeof window !== 'undefined' && window.localStorage) {
  try {
    isFailureEnabledState = localStorage.getItem(STORAGE_KEY) === 'true';
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
