import React, { useState, useEffect, useRef } from 'react';
import { Modal } from '../common/Modal';
import { Button } from '../common/Button';

export interface LogStreamEntry {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  raw?: string;
  fields?: Record<string, any>;
}

export interface LogStreamModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const LogStreamModal: React.FC<LogStreamModalProps> = ({ isOpen, onClose }) => {
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

  const renderFormattedPayload = (log: LogStreamEntry) => {
    let fieldsObj = log.fields;
    let parsedMessageObj: any = null;

    if (typeof log.message === 'string' && log.message.trim().startsWith('{')) {
      try {
        parsedMessageObj = JSON.parse(log.message);
      } catch {
        // not JSON
      }
    }

    if (parsedMessageObj) {
      return (
        <pre className="mt-2 p-2 bg-[#050811] text-[#4cd7f6] font-mono text-[11px] whitespace-pre-wrap break-words overflow-x-auto rounded-none">
          {JSON.stringify(parsedMessageObj, null, 2)}
        </pre>
      );
    }

    if (fieldsObj && Object.keys(fieldsObj).length > 0) {
      return (
        <pre className="mt-2 p-2 bg-[#050811] text-[#90b4ff] font-mono text-[11px] whitespace-pre-wrap break-words overflow-x-auto rounded-none">
          {JSON.stringify(fieldsObj, null, 2)}
        </pre>
      );
    }

    return null;
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Live Server Log Stream"
      subtitle="Real-time Zerolog log buffer streaming via GET /api/v1/logs/stream."
      icon="receipt_long"
      iconAccentColor="#4cd7f6"
      maxWidth="3xl"
      ariaLabelledBy="log-stream-modal-title"
      testId="log-stream-modal"
    >
      <div className="space-y-4 font-['Outfit']" data-testid="live-log-stream-card">
        <div className="flex items-center justify-between gap-4 flex-wrap pb-3 border-b border-[#464554]/30">
          <div className="flex items-center gap-2.5">
            <h3 className="text-sm font-bold text-white flex items-center gap-2">
              <span>Stream Status</span>
              {isLogStreamPaused ? (
                <span
                  className="px-2 py-0.5 text-[9px] font-mono font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40 rounded-none"
                  data-testid="log-stream-status-paused"
                >
                  PAUSED
                </span>
              ) : (
                <span
                  className="px-2 py-0.5 text-[9px] font-mono font-bold bg-emerald-500/20 text-emerald-300 border border-emerald-500/40 rounded-none flex items-center gap-1"
                  data-testid="log-stream-status-live"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping" />
                  STREAMING
                </span>
              )}
            </h3>
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
          className="p-1 bg-[#050810] rounded-none font-mono text-xs max-h-[28rem] overflow-y-auto space-y-1"
          data-testid="log-terminal-output"
        >
          {logs.length === 0 ? (
            <div className="text-[#717386] italic text-center py-8 text-[11px]" data-testid="log-terminal-empty">
              No server logs received yet. Listening to /api/v1/logs/stream...
            </div>
          ) : (
            logs.map((log, index) => {
              const style = getSeverityStyle(log.level);
              const formattedTime = log.timestamp?.includes('T')
                ? log.timestamp.split('T')[1].replace('Z', '')
                : log.timestamp;
              const isEven = index % 2 === 0;

              return (
                <div
                  key={log.id}
                  className={`p-2.5 rounded-none font-mono text-[11px] leading-relaxed transition-colors ${
                    isEven ? 'bg-[#0a0e19]' : 'bg-[#121828]'
                  }`}
                  data-testid="log-entry"
                >
                  <div className="flex items-start gap-2 flex-wrap">
                    <span className="text-[#717386] shrink-0 text-[10px] select-none">
                      [{formattedTime}]
                    </span>
                    <span className={`px-1.5 py-0.2 text-[9px] font-bold border rounded-none shrink-0 ${style.badge}`}>
                      {style.label}
                    </span>
                    <span className={`font-mono break-words whitespace-pre-wrap flex-1 min-w-[200px] ${style.text}`}>
                      {log.message}
                    </span>
                  </div>
                  {renderFormattedPayload(log)}
                </div>
              );
            })
          )}
          <div ref={terminalEndRef} />
        </div>
      </div>
    </Modal>
  );
};

export default LogStreamModal;
