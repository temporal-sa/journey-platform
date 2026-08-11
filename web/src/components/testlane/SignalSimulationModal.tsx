import React, { useState, useEffect } from 'react';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';
import { Badge } from '../common/Badge';
import { ToastMessageType } from '../common/Toast';
import { JourneyApiClient } from '../../api/client';

export interface SignalSimulationModalProps {
  isOpen: boolean;
  onClose: () => void;
  runId?: string;
  workflowId?: string;
  subjectId?: string;
  onFireToast?: (type: ToastMessageType, title: string, message: string) => void;
}

const apiClient = new JourneyApiClient();

const PRESET_PAYLOADS: Record<string, string> = {
  'order.completed': '{\n  "amount": 149.99,\n  "currency": "USD"\n}',
  'user_converted': '{\n  "conversion_type": "signup_upgrade",\n  "value": 99.00,\n  "plan": "pro_tier"\n}',
  'email_opened': '{\n  "email_id": "email-welcome-01",\n  "device": "mobile_ios",\n  "client": "apple_mail"\n}',
  'email_clicked': '{\n  "email_id": "email-welcome-01",\n  "link_url": "https://example.com/onboarding/start",\n  "target_button": "cta_primary"\n}',
  'cart_abandoned': '{\n  "cart_id": "cart-88421",\n  "item_count": 3,\n  "total_value": 219.50\n}',
};

export const SignalSimulationModal: React.FC<SignalSimulationModalProps> = ({
  isOpen,
  onClose,
  runId = 'run-601',
  workflowId: propWorkflowId = 'wf-welcome-series',
  subjectId: propSubjectId = 'usr_001',
  onFireToast,
}) => {
  const [eventType, setEventType] = useState<string>('order.completed');
  const [source, setSource] = useState<string>('web_control_center');
  const [schemaVersion, setSchemaVersion] = useState<string>('1.0');
  const [dataClassification, setDataClassification] = useState<'NonPII' | 'PII' | 'Sensitive'>('NonPII');
  const [subjectRef, setSubjectRef] = useState<string>(propSubjectId);
  const [targetWorkflowId, setTargetWorkflowId] = useState<string>(`wf-${propWorkflowId}-${runId}`);
  const [eventId, setEventId] = useState<string>('');
  const [traceId, setTraceId] = useState<string>(`trace-${runId}`);
  const [payloadJson, setPayloadJson] = useState<string>('{\n  "amount": 149.99,\n  "currency": "USD"\n}');
  const [isEmitting, setIsEmitting] = useState<boolean>(false);
  const [feedbackSuccess, setFeedbackSuccess] = useState<string | null>(null);
  const [feedbackError, setFeedbackError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setSubjectRef(propSubjectId || 'usr_001');
      setTargetWorkflowId(`wf-${propWorkflowId}-${runId}`);
      setTraceId(`trace-${runId}`);
      setEventId(`evt-sig-${Date.now().toString().slice(-4)}`);
      setFeedbackSuccess(null);
      setFeedbackError(null);
    }
  }, [isOpen, runId, propWorkflowId, propSubjectId]);

  const handleEmitEvent = async () => {
    setFeedbackError(null);
    setFeedbackSuccess(null);

    if (!eventType.trim()) {
      const err = 'Event Type (event_type) is required per schema.';
      setFeedbackError(err);
      if (onFireToast) onFireToast(ToastMessageType.ERROR, 'Missing Required Field', err);
      return;
    }

    let parsedData = {};
    try {
      if (payloadJson.trim()) {
        parsedData = JSON.parse(payloadJson);
      } else {
        const err = 'Event Data Payload (data) is required per schema.';
        setFeedbackError(err);
        if (onFireToast) onFireToast(ToastMessageType.ERROR, 'Missing Required Field', err);
        return;
      }
    } catch {
      const err = 'Please enter a valid JSON object payload.';
      setFeedbackError(err);
      if (onFireToast) onFireToast(ToastMessageType.ERROR, 'Invalid Event Payload JSON', err);
      return;
    }

    setIsEmitting(true);
    try {
      const generatedEvtId = eventId.trim() || `evt-sig-${Date.now().toString().slice(-6)}`;
      const generatedTraceId = traceId.trim() || `trace-${runId}-${Date.now()}`;

      await apiClient.emitKafkaTestEvent({
        schema_version: schemaVersion || '1.0',
        event_id: generatedEvtId,
        trace_id: generatedTraceId,
        event_type: eventType.trim(),
        source: source.trim() || 'web_control_center',
        subject: subjectRef.trim() || undefined,
        timestamp: new Date().toISOString(),
        data_classification: dataClassification,
        data: {
          ...parsedData,
          run_id: runId,
          ...(targetWorkflowId.trim() ? { workflow_id: targetWorkflowId.trim() } : {}),
        },
      });

      const successMsg = `Successfully emitted event '${eventType.trim()}' (${generatedEvtId}) to ingress topic & waiting workflows!`;
      setFeedbackSuccess(successMsg);

      if (onFireToast) {
        onFireToast(
          ToastMessageType.SUCCESS,
          'Signal Event Emitted Successfully',
          successMsg
        );
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to emit signal event.';
      setFeedbackError(errMsg);
      if (onFireToast) {
        onFireToast(
          ToastMessageType.ERROR,
          'Signal Event Emission Failed',
          errMsg
        );
      }
    } finally {
      setIsEmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Signal & Ingress Event Simulation"
      subtitle="Emit custom simulated signals directly to waiting workflow state machines and ingress event topic"
      icon="sensors"
      iconAccentColor="#4cd7f6"
      maxWidth="lg"
      testId="signal-simulation-modal"
      footer={
        <>
          <Button variant="secondary-dark" onClick={onClose} type="button">
            Close
          </Button>
          <Button
            type="button"
            variant="primary-teal"
            size="md"
            icon="sensors"
            onClick={handleEmitEvent}
            isLoading={isEmitting}
            data-testid="emit-signal-event-btn"
          >
            {isEmitting ? 'Emitting Signal...' : 'Emit Signal Event'}
          </Button>
        </>
      }
    >
      <div className="space-y-5 text-[#DFE2F1] font-['Outfit',sans-serif]" data-testid="signal-simulation-content">
        {feedbackSuccess && (
          <div role="status" className="p-3.5 bg-emerald-500/15 border border-emerald-500/40 text-emerald-300 text-xs font-['Outfit']">
            {feedbackSuccess}
          </div>
        )}

        {feedbackError && (
          <div role="alert" className="p-3.5 bg-rose-500/15 border border-rose-500/40 text-rose-300 text-xs font-['Outfit']">
            {feedbackError}
          </div>
        )}

        {/* Quick Signal Presets */}
        <div className="space-y-2 mb-4">
          <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider font-semibold font-['Outfit',sans-serif]">
            Quick Preset Signal Types
          </label>
          <div className="flex items-center gap-2 flex-wrap">
            {[
              { type: 'order.completed', label: 'Order Completed' },
              { type: 'user_converted', label: 'User Converted' },
              { type: 'email_opened', label: 'Email Opened' },
              { type: 'email_clicked', label: 'Email Clicked' },
              { type: 'cart_abandoned', label: 'Cart Abandoned' },
            ].map((preset) => (
              <Button
                key={preset.type}
                type="button"
                size="sm"
                variant={eventType === preset.type ? 'primary-cyan' : 'secondary-dark'}
                onClick={() => {
                  setEventType(preset.type);
                  if (PRESET_PAYLOADS[preset.type]) {
                    setPayloadJson(PRESET_PAYLOADS[preset.type]);
                  }
                }}
                className={`text-xs rounded-none font-['Outfit',sans-serif] ${
                  eventType === preset.type
                    ? 'bg-[#4cd7f6]/20 hover:bg-[#4cd7f6]/35 text-[#4cd7f6] border border-[#4cd7f6]/60 font-bold shadow-sm'
                    : 'bg-[#171b26] hover:bg-[#202533] text-[#908fa0] hover:text-white border border-[#464554]/60 font-medium'
                }`}
              >
                {preset.label}
              </Button>
            ))}
          </div>
        </div>
        {/* Main Form Parameters */}
        <div className="p-4 bg-[#090D16] space-y-4 rounded-none">
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold font-['Outfit',sans-serif]">
                Event Type <span className="text-rose-400">*</span>
              </label>
              <input
                type="text"
                value={eventType}
                onChange={(e) => setEventType(e.target.value)}
                placeholder="order.completed"
                data-testid="signal-event-type-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>

            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold font-['Outfit',sans-serif]">
                Source <span className="text-rose-400">*</span>
              </label>
              <input
                type="text"
                value={source}
                onChange={(e) => setSource(e.target.value)}
                placeholder="web_control_center"
                data-testid="signal-event-source-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>

            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold font-['Outfit',sans-serif]">
                Data Classification <span className="text-rose-400">*</span>
              </label>
              <select
                value={dataClassification}
                onChange={(e) => setDataClassification(e.target.value as 'NonPII' | 'PII' | 'Sensitive')}
                data-testid="signal-data-classification-select"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#4cd7f6] text-xs rounded-none focus:border-[#4cd7f6] transition-all cursor-pointer font-['Outfit',sans-serif]"
              >
                <option value="NonPII">NonPII</option>
                <option value="PII">PII</option>
                <option value="Sensitive">Sensitive</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold font-['Outfit',sans-serif]">
                Subject / Customer ID
              </label>
              <input
                type="text"
                value={subjectRef}
                onChange={(e) => setSubjectRef(e.target.value)}
                placeholder="usr_001"
                data-testid="signal-subject-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-white text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>

            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-semibold font-['Outfit',sans-serif]">
                Target Workflow ID
              </label>
              <input
                type="text"
                value={targetWorkflowId}
                onChange={(e) => setTargetWorkflowId(e.target.value)}
                placeholder="wf-draft-..."
                data-testid="signal-workflow-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#c0c1ff] text-xs rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-['Outfit',sans-serif]">
                Schema Version
              </label>
              <input
                type="text"
                value={schemaVersion}
                onChange={(e) => setSchemaVersion(e.target.value)}
                placeholder="1.0"
                data-testid="signal-schema-version-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] text-xs rounded-none outline-none font-['Outfit',sans-serif]"
              />
            </div>

            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-['Outfit',sans-serif]">
                Event ID
              </label>
              <input
                type="text"
                value={eventId}
                onChange={(e) => setEventId(e.target.value)}
                placeholder="evt-sig-..."
                data-testid="signal-event-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] text-xs rounded-none outline-none placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>

            <div>
              <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1 font-['Outfit',sans-serif]">
                Trace ID
              </label>
              <input
                type="text"
                value={traceId}
                onChange={(e) => setTraceId(e.target.value)}
                placeholder="trace-..."
                data-testid="signal-trace-id-input"
                style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
                className="w-full px-2.5 py-1.5 bg-[#111520] border border-[#464554] text-[#908fa0] text-xs rounded-none outline-none placeholder-[#464554] font-['Outfit',sans-serif]"
              />
            </div>
          </div>

          <div>
            <label className="block text-[10px] text-[#908fa0] uppercase tracking-wider mb-1.5 font-semibold font-['Outfit',sans-serif]">
              Event Payload Data (JSON) <span className="text-rose-400">*</span>
            </label>
            <textarea
              rows={3}
              value={payloadJson}
              onChange={(e) => setPayloadJson(e.target.value)}
              data-testid="signal-payload-textarea"
              style={{ background: '#111520', border: '1px solid #464554', outline: 'none' }}
              className="w-full px-3 py-2 bg-[#111520] border border-[#464554] text-[#4cd7f6] font-mono text-[11px] rounded-none focus:border-[#4cd7f6] transition-all placeholder-[#464554]"
            />
          </div>
        </div>
      </div>
    </Modal>
  );
};
