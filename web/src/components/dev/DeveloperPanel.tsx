import React, { useState, useEffect } from 'react';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { Badge } from '../common/Badge';
import { ToastMessageType } from '../common/Toast';
import { BUILD_TAG } from '../../buildTag';
import { useEditorStore } from '../../stores/editorStore';
import { apiClient } from '../../api/client';
import {
  isSimulatedApiFailureEnabled,
  toggleSimulatedApiFailure,
} from '../../api/simulatedFailure';

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
  const [eventType, setEventType] = useState<string>('order.completed');
  const [subjectRef, setSubjectRef] = useState<string>('usr_gold_varA_event');
  const [payloadJson, setPayloadJson] = useState<string>('{\n  "amount": 149.99,\n  "currency": "USD"\n}');
  const [isEmitting, setIsEmitting] = useState<boolean>(false);
  const currentDraft = useEditorStore((s) => s.currentDraft);
  useEffect(() => {
    if (isOpen) {
      setIsApiFailureActive(isSimulatedApiFailureEnabled());
    }
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
  const handleEmitEvent = async () => {
    let parsedData = {};
    try {
      if (payloadJson.trim()) {
        parsedData = JSON.parse(payloadJson);
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
      const eventId = `evt-dev-${Date.now().toString().slice(-6)}`;
      await apiClient.emitKafkaTestEvent({
        schema_version: '1.0',
        event_id: eventId,
        trace_id: `trace-${Date.now()}`,
        event_type: eventType || 'order.completed',
        source: 'dev_control_panel',
        subject: subjectRef || undefined,
        timestamp: new Date().toISOString(),
        data_classification: 'NonPII',
        data: parsedData,
      });

      onFireToast(
        ToastMessageType.SUCCESS,
        'Event Emitted Successfully',
        `Emitted event '${eventType}' (${eventId}) to ingress topic & waiting workflows.`
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
                  KAFKA / WEBMCP
                </span>
              </h3>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Emit custom inbound events (<code className="text-[#4cd7f6] bg-[#111520] px-1.5 py-0.5 border border-[#464554]/40 font-mono text-[10px]">POST /api/v1/events/emit</code>) to signal waiting workflows.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1.5 font-semibold">
                Event Type (Matches WaitForEvent node)
              </label>
              <input
                type="text"
                value={eventType}
                onChange={(e) => setEventType(e.target.value)}
                placeholder="order.completed"
                data-testid="dev-event-type-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-3 py-2 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>

            <div>
              <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1.5 font-semibold">
                Subject / Customer ID (Optional)
              </label>
              <input
                type="text"
                value={subjectRef}
                onChange={(e) => setSubjectRef(e.target.value)}
                placeholder="usr_gold_varA_event"
                data-testid="dev-subject-ref-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-3 py-2 bg-[#111520] border border-[#464554] text-white font-mono text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
              />
            </div>
          </div>

          <div>
            <label className="block text-[10px] font-mono text-[#908fa0] uppercase tracking-wider mb-1.5 font-semibold">
              Event Payload Attributes (JSON)
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
              variant="cyan"
              size="sm"
              icon="bolt"
              className="font-bold bg-[#4cd7f6]/20 hover:bg-[#4cd7f6]/35 text-[#4cd7f6] border border-[#4cd7f6]/50 shadow-sm rounded-none"
            >
              {isEmitting ? 'Emitting Event...' : 'Emit Ingress Event'}
            </Button>
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
