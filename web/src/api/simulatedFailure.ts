// Simulated API Failure Mode Helper & State

let isFailureEnabledState = false;
let isActivityFailureEnabledState = false;
let globalMaxFailureAttemptsState = 3;

const STORAGE_KEY = 'agy_simulated_api_failure_enabled';
const ACTIVITY_STORAGE_KEY = 'agy_simulated_activity_failure_enabled';
const MAX_ATTEMPTS_STORAGE_KEY = 'agy_simulated_activity_max_failure_attempts';

if (typeof window !== 'undefined' && window.localStorage) {
  try {
    isFailureEnabledState = localStorage.getItem(STORAGE_KEY) === 'true';
    isActivityFailureEnabledState = localStorage.getItem(ACTIVITY_STORAGE_KEY) === 'true';
    const storedMax = localStorage.getItem(MAX_ATTEMPTS_STORAGE_KEY);
    if (storedMax) {
      const parsed = parseInt(storedMax, 10);
      if (!isNaN(parsed) && parsed >= 1 && parsed <= 99) {
        globalMaxFailureAttemptsState = parsed;
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
 * Gets global maximum failure attempts for activity simulation.
 */
export function getGlobalMaxFailureAttempts(): number {
  return globalMaxFailureAttemptsState;
}

/**
 * Sets global maximum failure attempts for activity simulation (min 1, max 10, default 3).
 */
export function setGlobalMaxFailureAttempts(attempts: number): void {
  const clamped = Math.min(99, Math.max(1, attempts));
  globalMaxFailureAttemptsState = clamped;
  if (typeof window !== 'undefined' && window.localStorage) {
    try {
      localStorage.setItem(MAX_ATTEMPTS_STORAGE_KEY, String(clamped));
    } catch {
      // Ignore
    }
  }
}
