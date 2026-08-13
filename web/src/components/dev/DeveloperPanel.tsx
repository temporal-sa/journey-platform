import React, { useState, useEffect, useRef } from 'react';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { Badge } from '../common/Badge';
import { ToastMessageType } from '../common/Toast';
import { BUILD_TAG } from '../../buildTag';
import { useEditorStore } from '../../stores/editorStore';
import { apiClient } from '../../api/client';
import { WorkerStatusResponse } from '../../types/api';
import {
  isSimulatedApiFailureEnabled,
  toggleSimulatedApiFailure,
  isSimulatedActivityFailureEnabled,
  toggleSimulatedActivityFailure,
  getGlobalMaxFailureAttempts,
  setGlobalMaxFailureAttempts,
} from '../../api/simulatedFailure';
export interface LogStreamEntry {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  raw?: string;
  fields?: Record<string, any>;
}

export interface DeveloperPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onFireToast: (type: ToastMessageType, title: string, message: string) => void;
}

export const DeveloperPanel: React.FC<DeveloperPanelProps> = ({
  isOpen,
  onClose,
  onFireToast,
}) => {
  const [isApiFailureActive, setIsApiFailureActive] = useState<boolean>(isSimulatedApiFailureEnabled());
  const [isActivityFailureActive, setIsActivityFailureActive] = useState<boolean>(isSimulatedActivityFailureEnabled());
  const [maxFailureAttempts, setMaxFailureAttempts] = useState<number>(getGlobalMaxFailureAttempts());
  const [eventType, setEventType] = useState<string>('order.completed');
  const [source, setSource] = useState<string>('web_control_panel');
  const [schemaVersion, setSchemaVersion] = useState<string>('1.0');
  const [dataClassification, setDataClassification] = useState<'NonPII' | 'PII' | 'Sensitive'>('NonPII');
  const [subjectRef, setSubjectRef] = useState<string>('usr_gold_varA_event');
  const [workflowId, setWorkflowId] = useState<string>('');
  const [eventId, setEventId] = useState<string>('');
  const [traceId, setTraceId] = useState<string>('');
  const [payloadJson, setPayloadJson] = useState<string>('{\n  "amount": 149.99,\n  "currency": "USD"\n}');
  const [isEmitting, setIsEmitting] = useState<boolean>(false);
  const [workerStatus, setWorkerStatus] = useState<WorkerStatusResponse>({
    status: 'stopped',
    pid: 0,
    uptime_seconds: 0,
  });
  const [isTogglingWorker, setIsTogglingWorker] = useState<boolean>(false);
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const [logs, setLogs] = useState<LogStreamEntry[]>([]);
  const [isLogStreamPaused, setIsLogStreamPaused] = useState<boolean>(false);
  const isPausedRef = useRef<boolean>(isLogStreamPaused);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    isPausedRef.current = isLogStreamPaused;
  }, [isLogStreamPaused]);

  useEffect(() => {
    if (!isOpen) return;

    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource('/api/v1/logs/stream');

      eventSource.onmessage = (event) => {
        if (isPausedRef.current) return;
        try {
          const data = JSON.parse(event.data);
          const newEntry: LogStreamEntry = {
            id: `log-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
            timestamp: data.timestamp || new Date().toISOString(),
            level: (data.level || 'info').toLowerCase(),
            message: data.message || data.raw || JSON.stringify(data),
            fields: data.fields,
            raw: data.raw,
          };
          setLogs((prev) => [...prev.slice(-499), newEntry]);
        } catch {
          // ignore parse errors
        }
      };
    } catch (e) {
      console.error('Failed to connect log stream', e);
    }

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [isOpen]);

  useEffect(() => {
    if (!isLogStreamPaused && terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, isLogStreamPaused]);

  const getSeverityStyle = (level: string) => {
    const l = (level || '').toLowerCase();
    if (l === 'error' || l === 'fatal' || l === 'panic') {
      return {
        badge: 'bg-rose-500/20 text-rose-300 border-rose-500/40',
        text: 'text-rose-400',
        label: 'ERROR',
      };
    }
    if (l === 'warn' || l === 'warning') {
      return {
        badge: 'bg-amber-500/20 text-amber-300 border-amber-500/40',
        text: 'text-amber-400',
        label: 'WARN',
      };
    }
    return {
      badge: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/40',
      text: 'text-cyan-400',
      label: 'INFO',
    };
  };
  useEffect(() => {
    if (isOpen) {
      setIsApiFailureActive(isSimulatedApiFailureEnabled());
      apiClient
        .getWorkerStatus()
        .then((res) => {
          if (res) setWorkerStatus(res);
        })
        .catch(() => {});
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen || typeof EventSource === 'undefined') return;

    const eventSource = new EventSource('/api/v1/worker/status/stream');

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data && typeof data.status === 'string') {
          setWorkerStatus(data);
        }
      } catch {
        // SSE parse error ignored
      }
    };

    return () => {
      eventSource.close();
    };
  }, [isOpen]);

  const handleToggleApiFailure = () => {
    const newState = toggleSimulatedApiFailure();
    setIsApiFailureActive(newState);
    if (newState) {
      onFireToast(
        ToastMessageType.WARNING,
        'API Failure Mode Enabled',
        'All client API requests will simulate HTTP 503 service unavailable errors.'
      );
    } else {
      onFireToast(
        ToastMessageType.INFO,
        'API Failure Mode Disabled',
        'Normal backend client network requests restored.'
      );
    }
  };

  const handleToggleActivityFailure = () => {
    const newState = toggleSimulatedActivityFailure();
    setIsActivityFailureActive(newState);
    if (newState) {
      setMaxFailureAttempts(99);
      setGlobalMaxFailureAttempts(99);
      onFireToast(
        ToastMessageType.WARNING,
        'Activity Failure Simulation Enabled',
        'Backend activities will simulate failure retries up to 99 attempts.'
      );
    } else {
      onFireToast(
        ToastMessageType.INFO,
        'Activity Failure Simulation Disabled',
        'Normal backend activity execution restored.'
      );
    }
  };

  const handleMaxFailureAttemptsChange = (val: number) => {
    const clamped = Math.min(99, Math.max(1, val));
    setMaxFailureAttempts(clamped);
    setGlobalMaxFailureAttempts(clamped);
  };
  const handleToggleWorker = async () => {
    setIsTogglingWorker(true);
    try {
      if (workerStatus.status === 'running') {
        const res = await apiClient.stopWorker();
        setWorkerStatus(res);
        onFireToast(
          ToastMessageType.INFO,
          'Temporal Worker Stopped',
          'journey-worker process stopped cleanly.'
        );
      } else {
        const res = await apiClient.startWorker();
        setWorkerStatus(res);
        onFireToast(
          ToastMessageType.SUCCESS,
          'Temporal Worker Started',
          `journey-worker process spawned with PID ${res.pid}.`
        );
      }
    } catch (err: any) {
      onFireToast(
        ToastMessageType.ERROR,
        'Worker Control Failed',
        err?.message || 'Failed to toggle journey-worker process.'
      );
    } finally {
      setIsTogglingWorker(false);
    }
  };
  const handleEmitEvent = async () => {
    if (!eventType.trim()) {
      onFireToast(
        ToastMessageType.ERROR,
        'Missing Required Field',
        'Event Type (event_type) is required per schema.'
      );
      return;
    }

    if (!source.trim()) {
      onFireToast(
        ToastMessageType.ERROR,
        'Missing Required Field',
        'Event Source (source) is required per schema.'
      );
      return;
    }

    let parsedData = {};
    try {
      if (payloadJson.trim()) {
        parsedData = JSON.parse(payloadJson);
      } else {
        onFireToast(
          ToastMessageType.ERROR,
          'Missing Required Field',
          'Event Data Payload (data) is required per schema.'
        );
        return;
      }
    } catch {
      onFireToast(
        ToastMessageType.ERROR,
        'Invalid Event Payload JSON',
        'Please enter a valid JSON object payload.'
      );
      return;
    }

    setIsEmitting(true);
    try {
      const generatedEvtId = eventId.trim() || `evt-dev-${Date.now().toString().slice(-6)}`;
      const generatedTraceId = traceId.trim() || `trace-${Date.now()}`;

      await apiClient.emitKafkaTestEvent({
        schema_version: (schemaVersion || '1.0') as '1.0',
        event_id: generatedEvtId,
        trace_id: generatedTraceId,
        event_type: eventType.trim(),
        source: source.trim(),
        subject: subjectRef.trim() || undefined,
        timestamp: new Date().toISOString(),
        data_classification: dataClassification,
        data: {
          ...parsedData,
          ...(workflowId.trim() ? { workflow_id: workflowId.trim() } : {}),
        },
      });

      onFireToast(
        ToastMessageType.SUCCESS,
        'Event Emitted Successfully',
        `Emitted event '${eventType}' (${generatedEvtId}) to ingress topic & waiting workflows.`
      );
    } catch (err: any) {
      onFireToast(
        ToastMessageType.ERROR,
        'Event Emission Failed',
        err?.message || 'Failed to emit event.'
      );
    } finally {
      setIsEmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Developer Control Panel"
      subtitle="Interactive diagnostic tools, API failure simulation, and toast notification sandbox"
      icon="developer_mode"
      iconAccentColor="#b76dff"
      badge={
        <Badge variant="purple" testId="dev-panel-badge">
          DEV TOOLS
        </Badge>
      }
      maxWidth="lg"
      testId="developer-panel-modal"
    >
      <div className="space-y-6 text-[#DFE2F1] font-['Outfit',sans-serif]" data-testid="developer-panel">
        {/* Section 1: API Failure Mode Simulation */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3.5 shadow-md">
          <div className="flex items-center justify-between gap-4 flex-wrap pb-3 border-b border-[#464554]/30">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-none bg-rose-500/10 border border-rose-500/30 flex items-center justify-center text-rose-400">
                <span className="material-symbols-outlined text-lg">cloud_off</span>
              </div>
              <div>
                <h3 className="text-sm font-bold text-white font-['Outfit'] flex items-center gap-2">
                  <span>API Failure Mode Simulation</span>
                </h3>
                <p className="text-xs text-[#908fa0] mt-0.5">
                  Simulate offline network states or HTTP 503 Service Unavailable API responses.
                </p>
              </div>
            </div>

            {/* Status Badge */}
            {isApiFailureActive ? (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40 rounded-none animate-pulse flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-rose-400 animate-ping" />
                ACTIVE
              </span>
            ) : (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-[#171b26] text-[#908fa0] border border-[#464554]/50 rounded-none">
                DISABLED
              </span>
            )}
          </div>

          <div className="flex items-center justify-between gap-4">
            <p className="text-xs text-[#c7c4d7] leading-relaxed">
              When enabled, all <code className="text-[#ddb7ff] bg-[#1a1d28] px-1.5 py-0.5 border border-[#464554]/40 font-mono text-[11px]">JourneyApiClient</code> network requests immediately fail with HTTP 503 status to verify error toast handling across draft saving, workflow publishing, static list upload, and test execution.
            </p>

            <Button
              type="button"
              onClick={handleToggleApiFailure}
              data-testid="toggle-api-failure-btn"
              variant={isApiFailureActive ? 'danger' : 'secondary-dark'}
              size="sm"
              icon={isApiFailureActive ? 'cloud_off' : 'cloud_done'}
              className={`shrink-0 font-bold ${
                isApiFailureActive
                  ? 'bg-rose-500/20 hover:bg-rose-500/35 text-rose-300 border border-rose-500/50 shadow-sm'
                  : 'bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554]'
              }`}
            >
              {isApiFailureActive ? 'Disable Failure Mode' : 'Enable Failure Mode'}
            </Button>
          </div>
        </div>
        {/* Section 1b: Activity Failure Simulation Mode */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3.5 shadow-md" data-testid="activity-failure-simulation-card">
          <div className="flex items-center justify-between gap-4 flex-wrap pb-3 border-b border-[#464554]/30">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-none bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400">
                <span className="material-symbols-outlined text-lg">sync_problem</span>
              </div>
              <div>
                <h3 className="text-sm font-bold text-white font-['Outfit'] flex items-center gap-2">
                  <span>Activity Failure Simulation Mode</span>
                </h3>
                <p className="text-xs text-[#908fa0] mt-0.5">
                  Simulate activity execution retries and fault injection across backend activities.
                </p>
              </div>
            </div>

            {/* Status Badge */}
            {isActivityFailureActive ? (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-none animate-pulse flex items-center gap-1.5">
                <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-ping" />
                ACTIVE
              </span>
            ) : (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-[#171b26] text-[#908fa0] border border-[#464554]/50 rounded-none">
                DISABLED
              </span>
            )}
          </div>

          <div className="flex items-center justify-between gap-4 flex-wrap">
            <p className="text-xs text-[#c7c4d7] leading-relaxed flex-1">
              When enabled, backend activities simulate execution failures up to max retry attempts so Temporal retries or records activity failure.
            </p>

            <Button
              type="button"
              onClick={handleToggleActivityFailure}
              data-testid="toggle-activity-failure-btn"
              variant={isActivityFailureActive ? 'warning' : 'secondary-dark'}
              size="sm"
              icon={isActivityFailureActive ? 'sync_problem' : 'play_circle'}
              className={`shrink-0 font-bold ${
                isActivityFailureActive
                  ? 'bg-amber-500/20 hover:bg-amber-500/35 text-amber-300 border border-amber-500/50 shadow-sm'
                  : 'bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554]'
              }`}
            >
              {isActivityFailureActive ? 'Disable Activity Simulation' : 'Enable Activity Simulation'}
            </Button>
          </div>

          <div className="pt-2 border-t border-[#464554]/30 flex items-center justify-between gap-4">
            <label className="text-xs font-medium text-[#c7c4d7]">
              Max Failure Retry Attempts (1-99)
            </label>
            <input
              type="number"
              min={1}
              max={99}
              value={maxFailureAttempts}
              onChange={(e) => handleMaxFailureAttemptsChange(e.target.value === '' ? 99 : Number(e.target.value))}
              data-testid="dev-max-failure-attempts-input"
              className="w-24 px-3 py-1 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#ddb7ff] outline-none"
            />
          </div>
        </div>
        {/* Section: Temporal Worker Process Control */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3.5 shadow-md" data-testid="worker-control-card">
          <div className="flex items-center justify-between gap-4 flex-wrap pb-3 border-b border-[#464554]/30">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-none bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
                <span className="material-symbols-outlined text-lg">memory</span>
              </div>
              <div>
                <h3 className="text-sm font-bold text-white font-['Outfit'] flex items-center gap-2">
                  <span>Temporal Worker Process Control</span>
                </h3>
                <p className="text-xs text-[#908fa0] mt-0.5">
                  Start, stop, and monitor the local <code className="text-[#ddb7ff] bg-[#1a1d28] px-1.5 py-0.5 border border-[#464554]/40 font-mono text-[11px]">journey-worker</code> daemon process.
                </p>
              </div>
            </div>

            {/* Status Badge */}
            {workerStatus.status === 'running' ? (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 rounded-none flex items-center gap-1.5" data-testid="worker-status-badge">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping" />
                RUNNING
              </span>
            ) : (
              <span className="px-2.5 py-1 text-[10px] font-mono font-bold bg-rose-500/20 text-rose-300 border border-rose-500/40 rounded-none flex items-center gap-1.5" data-testid="worker-status-badge">
                <span className="w-1.5 h-1.5 rounded-full bg-rose-400" />
                STOPPED
              </span>
            )}
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="text-xs font-mono text-[#c7c4d7] space-y-1">
              <div>
                Status: <span className={workerStatus.status === 'running' ? 'text-emerald-400 font-bold' : 'text-rose-400 font-bold'}>{workerStatus.status.toUpperCase()}</span>
                {workerStatus.status === 'running' && (
                  <span className="ml-3 text-[#908fa0]">
                    PID: <span className="text-[#c0c1ff]">{workerStatus.pid}</span> | Uptime: <span className="text-[#c0c1ff]">{workerStatus.uptime_seconds}s</span>
                  </span>
                )}
              </div>
            </div>

            <Button
              type="button"
              onClick={handleToggleWorker}
              disabled={isTogglingWorker}
              data-testid="toggle-worker-btn"
              variant={workerStatus.status === 'running' ? 'danger' : 'secondary-dark'}
              size="sm"
              icon={workerStatus.status === 'running' ? 'stop' : 'play_arrow'}
              className={`shrink-0 font-bold ${
                workerStatus.status === 'running'
                  ? 'bg-rose-500/20 hover:bg-rose-500/35 text-rose-300 border border-rose-500/50 shadow-sm'
                  : 'bg-emerald-500/20 hover:bg-emerald-500/35 text-emerald-300 border border-emerald-500/50 shadow-sm'
              }`}
            >
              {isTogglingWorker
                ? (workerStatus.status === 'running' ? 'Stopping Worker...' : 'Starting Worker...')
                : (workerStatus.status === 'running' ? 'Stop Worker' : 'Start Worker')}
            </Button>
          </div>
        </div>

        {/* Section 2: Toast Notification Sandbox */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3.5 shadow-md">
          <div className="flex items-center gap-2.5 pb-3 border-b border-[#464554]/30">
            <div className="w-8 h-8 rounded-none bg-[#b76dff]/10 border border-[#b76dff]/30 flex items-center justify-center text-[#ddb7ff]">
              <span className="material-symbols-outlined text-lg">notifications_active</span>
            </div>
            <div>
              <h3 className="text-sm font-bold text-white font-['Outfit']">Toast Notification Sandbox</h3>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Interactively trigger non-blocking toast notifications across all 4 standardized status variants.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Success Toast Trigger */}
            <Button
              type="button"
              onClick={() => {
                onFireToast(
                  ToastMessageType.SUCCESS,
                  'Journey Execution Succeeded',
                  'Test run scenario executed cleanly across 4 mock nodes.'
                );
              }}
              data-testid="dev-toast-success"
              variant="secondary-dark"
              size="md"
              icon="check_circle"
              fullWidth
              className="justify-start bg-[#0F131D] hover:bg-emerald-500/10 border-emerald-500/30 text-emerald-300"
            >
              <div className="text-left min-w-0">
                <div className="text-xs font-bold text-white">Fire Success Toast</div>
                <div className="text-[10px] text-emerald-400/80 font-mono mt-0.5">ToastMessageType.SUCCESS</div>
              </div>
            </Button>

            {/* Error Toast Trigger */}
            <Button
              type="button"
              onClick={() => {
                onFireToast(
                  ToastMessageType.ERROR,
                  'Workflow Execution Failed',
                  'State transition error on node-301: Connection timed out.'
                );
              }}
              data-testid="dev-toast-error"
              variant="danger"
              size="md"
              icon="error"
              fullWidth
              className="justify-start bg-[#0F131D] hover:bg-rose-500/20 border-rose-500/40 text-rose-200"
            >
              <div className="text-left min-w-0">
                <div className="text-xs font-bold text-white">Fire Error Toast</div>
                <div className="text-[10px] text-rose-300/80 font-mono mt-0.5">ToastMessageType.ERROR</div>
              </div>
            </Button>

            {/* Warning Toast Trigger */}
            <Button
              type="button"
              onClick={() => {
                onFireToast(
                  ToastMessageType.WARNING,
                  'Rate Limit Approaching',
                  'Provider mock throughput is operating at 92% capacity.'
                );
              }}
              data-testid="dev-toast-warning"
              variant="secondary-dark"
              size="md"
              icon="warning"
              fullWidth
              className="justify-start bg-[#0F131D] hover:bg-amber-500/10 border-amber-500/30 text-amber-300"
            >
              <div className="text-left min-w-0">
                <div className="text-xs font-bold text-white">Fire Warning Toast</div>
                <div className="text-[10px] text-amber-400/80 font-mono mt-0.5">ToastMessageType.WARNING</div>
              </div>
            </Button>

            {/* Info Toast Trigger */}
            <Button
              type="button"
              onClick={() => {
                onFireToast(
                  ToastMessageType.INFO,
                  'System State Synced',
                  'ETag draft hash updated to draft-101-v5 revision.'
                );
              }}
              data-testid="dev-toast-info"
              variant="secondary-dark"
              size="md"
              icon="info"
              fullWidth
              className="justify-start bg-[#0F131D] hover:bg-cyan-500/10 border-cyan-500/30 text-cyan-300"
            >
              <div className="text-left min-w-0">
                <div className="text-xs font-bold text-white">Fire Info Toast</div>
                <div className="text-[10px] text-cyan-400/80 font-mono mt-0.5">ToastMessageType.INFO</div>
              </div>
            </Button>
          </div>
        </div>
        {/* Section 3: Interactive Event Ingress & WebMCP Signal Simulator */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-4 shadow-md">
          <div className="flex items-center gap-2.5 pb-3 border-b border-[#464554]/30">
            <div className="w-8 h-8 rounded-none bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 flex items-center justify-center text-[#4cd7f6] shrink-0">
              <span className="material-symbols-outlined text-lg">electric_bolt</span>
            </div>
            <div>
              <h3 className="text-sm font-bold text-white font-['Outfit'] flex items-center gap-2">
                <span>Event Ingress & Signal Simulator</span>
                <span className="px-2 py-0.5 text-[9px] font-mono font-bold bg-[#4cd7f6]/15 text-[#4cd7f6] border border-[#4cd7f6]/30 rounded-none">
                  EVENT ENVELOPE v1.0
                </span>
              </h3>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Emit custom inbound events (<code className="text-[#4cd7f6] bg-[#111520] px-1.5 py-0.5 border border-[#464554]/40 font-mono text-[10px]">POST /api/v1/events/emit</code>) with all required schema envelope attributes.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
                Event Type <span className="text-rose-400">*</span>
              </label>
              <input
                type="text"
                value={eventType}
                onChange={(e) => setEventType(e.target.value)}
                placeholder="order.completed"
                data-testid="dev-event-type-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
                Source <span className="text-rose-400">*</span>
              </label>
              <input
                type="text"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                placeholder="web_control_panel"
                data-testid="dev-event-source-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
                Data Classification <span className="text-rose-400">*</span>
              </label>
              <select
                value={dataClassification}
                onChange={(e) => setDataClassification(e.target.value as 'NonPII' | 'PII' | 'Sensitive')}
                data-testid="dev-data-classification-select"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#4cd7f6] font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all cursor-pointer"
              >
                <option value="NonPII">NonPII</option>
                <option value="PII">PII</option>
                <option value="Sensitive">Sensitive</option>
              </select>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
                Subject / Customer ID (Optional)
              </label>
              <input
                type="text"
                value={subjectRef}
                onChange={(e) => setSubjectRef(e.target.value)}
                placeholder="usr_gold_varA_event"
                data-testid="dev-subject-ref-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1 font-semibold">
                Target Workflow ID (Optional)
              </label>
              <input
                type="text"
                value={workflowId}
                onChange={(e) => setWorkflowId(e.target.value)}
                placeholder="wf-draft-1786135080506-..."
                data-testid="dev-workflow-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#c0c1ff] font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1">
                Schema Version
              </label>
              <input
                type="text"
                value={schemaVersion}
                onChange={(e) => setSchemaVersion(e.target.value)}
                placeholder="1.0"
                data-testid="dev-schema-version-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] font-mono text-xs rounded-none outline-none"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1">
                Event ID (Auto-generated if blank)
              </label>
              <input
                type="text"
                value={eventId}
                onChange={(e) => setEventId(e.target.value)}
                placeholder="evt-dev-..."
                data-testid="dev-event-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] font-mono text-xs rounded-none outline-none placeholder-[#464554]"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1">
                Trace ID (Auto-generated if blank)
              </label>
              <input
                type="text"
                value={traceId}
                onChange={(e) => setTraceId(e.target.value)}
                placeholder="trace-..."
                data-testid="dev-trace-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] font-mono text-xs rounded-none outline-none placeholder-[#464554]"
              />
            </div>
          </div>

          <div>
            <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1.5 font-semibold">
              Event Payload Data (JSON) <span className="text-rose-400">*</span>
            </label>
            <textarea
              rows={3}
              value={payloadJson}
              onChange={(e) => setPayloadJson(e.target.value)}
              data-testid="dev-event-payload-textarea"
              style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
              className="w-full px-3 py-2 bg-[#111520] border border-[#464554] text-[#4cd7f6] font-mono text-[11px] rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
            />
          </div>

          <div className="flex justify-between items-center pt-1">
            <span className="text-[10px] font-mono text-[#908fa0]">
              Endpoint: <span className="text-[#c0c1ff]">/api/v1/events/emit</span>
            </span>

            <Button
              type="button"
              onClick={handleEmitEvent}
              disabled={isEmitting}
              data-testid="emit-ingress-event-btn"
              variant="primary-cyan"
              size="sm"
              icon="bolt"
              className="font-bold bg-[#4cd7f6]/20 hover:bg-[#4cd7f6]/35 text-[#4cd7f6] border border-[#4cd7f6]/50 shadow-sm rounded-none"
            >
              {isEmitting ? 'Emitting Event...' : 'Emit Ingress Event'}
            </Button>
          </div>
        </div>
        {/* Section: Live Server Log Stream */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3.5 shadow-md" data-testid="live-log-stream-card">
          <div className="flex items-center justify-between gap-4 flex-wrap pb-3 border-b border-[#464554]/30">
            <div className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-none bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 flex items-center justify-center text-[#4cd7f6]">
                <span className="material-symbols-outlined text-lg">receipt_long</span>
              </div>
              <div>
                <h3 className="text-sm font-bold text-white font-['Outfit'] flex items-center gap-2">
                  <span>Live Server Log Stream</span>
                  {isLogStreamPaused ? (
                    <span className="px-2 py-0.5 text-[9px] font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-none" data-testid="log-stream-status-paused">
                      PAUSED
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 text-[9px] font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 rounded-none flex items-center gap-1" data-testid="log-stream-status-live">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping" />
                      STREAMING
                    </span>
                  )}
                </h3>
                <p className="text-xs text-[#908fa0] mt-0.5">
                  Real-time Zerolog log buffer streaming via <code className="text-[#4cd7f6] bg-[#111520] px-1.5 py-0.5 border border-[#464554]/40 font-mono text-[10px]">GET /api/v1/logs/stream</code>.
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Button
                type="button"
                onClick={() => setIsLogStreamPaused(!isLogStreamPaused)}
                data-testid="toggle-log-stream-btn"
                variant="secondary-dark"
                size="sm"
                icon={isLogStreamPaused ? 'play_arrow' : 'pause'}
                className={`font-bold rounded-none ${
                  isLogStreamPaused
                    ? 'bg-emerald-500/20 hover:bg-emerald-500/35 text-emerald-300 border border-emerald-500/50'
                    : 'bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554]'
                }`}
              >
                {isLogStreamPaused ? 'Resume Stream' : 'Pause Stream'}
              </Button>

              <Button
                type="button"
                onClick={() => setLogs([])}
                data-testid="clear-logs-btn"
                variant="secondary-dark"
                size="sm"
                icon="delete"
                className="font-bold bg-[#1c1f2a] hover:bg-rose-500/20 text-[#dfe2f1] hover:text-rose-300 border border-[#464554] rounded-none"
              >
                Clear Logs
              </Button>
            </div>
          </div>

          <div
            className="p-3 bg-[#050810] border border-[#464554]/50 rounded-none font-mono text-xs max-h-64 overflow-y-auto space-y-1.5"
            data-testid="log-terminal-output"
          >
            {logs.length === 0 ? (
              <div className="text-[#717386] italic text-center py-6 text-[11px]" data-testid="log-terminal-empty">
                No server logs received yet. Listening to /api/v1/logs/stream...
              </div>
            ) : (
              logs.map((log) => {
                const style = getSeverityStyle(log.level);
                const formattedTime = log.timestamp?.includes('T')
                  ? log.timestamp.split('T')[1].replace('Z', '')
                  : log.timestamp;

                return (
                  <div
                    key={log.id}
                    className="flex items-start gap-2 leading-relaxed border-b border-[#1c2233]/40 pb-1 text-[11px]"
                    data-testid="log-entry"
                  >
                    <span className="text-[#717386] shrink-0 text-[10px] select-none">
                      [{formattedTime}]
                    </span>
                    <span className={`px-1.5 py-0.2 text-[9px] font-bold border rounded-none shrink-0 ${style.badge}`}>
                      {style.label}
                    </span>
                    <span className={`font-mono break-all ${style.text}`}>
                      {log.message}
                    </span>
                    {log.fields && Object.keys(log.fields).length > 0 && (
                      <span className="text-[#717386] text-[10px] truncate max-w-xs">
                        {JSON.stringify(log.fields)}
                      </span>
                    )}
                  </div>
                );
              })
            )}
            <div ref={terminalEndRef} />
          </div>
        </div>
        {/* Section 3: Runtime Diagnostics & System Metadata */}
        <div className="p-4 bg-[#090D16] border border-[#464554]/60 rounded-none space-y-3 shadow-md">
          <div className="flex items-center gap-2.5 pb-2 border-b border-[#464554]/30">
            <div className="w-8 h-8 rounded-none bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 flex items-center justify-center text-[#4cd7f6]">
              <span className="material-symbols-outlined text-lg">terminal</span>
            </div>
            <div>
              <h3 className="text-sm font-bold text-white font-['Outfit']">Runtime Metadata & Diagnostics</h3>
              <p className="text-xs text-[#908fa0] mt-0.5">Current environment state and active workspace parameters.</p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs font-mono">
            <div className="p-2.5 bg-[#111520] border border-[#464554]/40 flex flex-col justify-between gap-1">
              <span className="text-[#908fa0] text-[10px] uppercase tracking-wider">Build Tag</span>
              <span className="text-[#c0c1ff] font-bold truncate">{BUILD_TAG}</span>
            </div>

            <div className="p-2.5 bg-[#111520] border border-[#464554]/40 flex flex-col justify-between gap-1">
              <span className="text-[#908fa0] text-[10px] uppercase tracking-wider">Developer Shortcut</span>
              <div className="flex items-center gap-1.5">
                <kbd className="px-1.5 py-0.5 bg-[#1F2433] border border-[#464554] text-[#4cd7f6] text-[11px] rounded-none">Cmd + ?</kbd>
                <span className="text-[#908fa0] text-[10px]">or</span>
                <kbd className="px-1.5 py-0.5 bg-[#1F2433] border border-[#464554] text-[#4cd7f6] text-[11px] rounded-none">Ctrl + ?</kbd>
              </div>
            </div>

            <div className="p-2.5 bg-[#111520] border border-[#464554]/40 flex flex-col justify-between gap-1">
              <span className="text-[#908fa0] text-[10px] uppercase tracking-wider">Active Journey ID</span>
              <span className="text-[#dfe2f1] font-bold truncate">{currentDraft?.draft_id || 'draft-101'}</span>
            </div>

            <div className="p-2.5 bg-[#111520] border border-[#464554]/40 flex flex-col justify-between gap-1">
              <span className="text-[#908fa0] text-[10px] uppercase tracking-wider">Active Journey Name</span>
              <span className="text-[#dfe2f1] font-bold truncate">{currentDraft?.name || 'New Journey'}</span>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default DeveloperPanel;
