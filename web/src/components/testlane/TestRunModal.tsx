import { useState, useEffect } from 'react';
import { JourneyApiClient } from '../../api/client';
import { Button } from '../common/Button';
import { Modal } from '../common/Modal';
import { Badge } from '../common/Badge';
import type { TestRun, StaticList } from '../../types/api';

const apiClient = new JourneyApiClient();

export interface TestRunModalProps {
  isOpen: boolean;
  onClose: () => void;
  onStartTestRun?: (config: TestRunConfig) => void;
  draftId?: string;
}

export interface TestRunConfig {
  draftId: string;
  executionMode: 'realistic' | 'forced_variant_coverage';
  fakeProviders: string[];
  expiryHours: number;
  staticListId?: string;
}

export function TestRunModal({
  isOpen,
  onClose,
  onStartTestRun,
  draftId = 'draft-101',
}: TestRunModalProps) {
  const [executionMode, setExecutionMode] = useState<'realistic' | 'forced_variant_coverage'>('realistic');
  const [fakeProviders, setFakeProviders] = useState<string[]>(['mock-sendgrid', 'mock-twilio']);
  const [expiryHours, setExpiryHours] = useState('24');
  const [staticLists, setStaticLists] = useState<StaticList[]>([]);
  const [selectedStaticListId, setSelectedStaticListId] = useState<string>('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  useEffect(() => {
    if (isOpen) {
      apiClient
        .listStaticLists()
        .then((lists) => {
          if (Array.isArray(lists)) {
            setStaticLists(lists);
            if (lists.length > 0 && !selectedStaticListId) {
              setSelectedStaticListId(lists[0].list_id);
            }
          }
        })
        .catch((err) => console.error('Failed to fetch static lists:', err));
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const toggleProvider = (provider: string) => {
    setFakeProviders((prev) =>
      prev.includes(provider) ? prev.filter((p) => p !== provider) : [...prev, provider]
    );
  };

  const handleRunSubmit = async () => {
    if (!selectedStaticListId) {
      setValidationError('Please select a static audience list to execute a test run.');
      return;
    }
    setValidationError(null);
    setIsSubmitting(true);
    try {
      const payload: Partial<TestRun> = {
        schema_version: '1.0',
        test_run_id: `tr-${Date.now().toString().slice(-6)}`,
        draft_id: draftId,
        static_list_id: selectedStaticListId,
        mock_inputs: {
          execution_mode: executionMode,
          fake_providers: fakeProviders,
          expiry_hours: parseInt(expiryHours, 10),
          static_list_id: selectedStaticListId,
        },
      };
      if (onStartTestRun) {
        await onStartTestRun({
          draftId,
          executionMode,
          fakeProviders,
          expiryHours: parseInt(expiryHours, 10),
          staticListId: selectedStaticListId,
        });
      } else {
        await apiClient.startTestRun(payload);
      }
      onClose();
    } catch (err) {
      console.error('Failed to launch test run:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Configure Test Run Execution"
      icon="play_circle"
      iconAccentColor="#4cd7f6"
      maxWidth="2xl"
      testId="test-run-modal"
      ariaLabelledBy="test-run-modal-title"
      footer={
        <>
          <Button variant="secondary-dark" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary-teal"
            size="lg"
            onClick={handleRunSubmit}
            data-testid="start-test-run-btn"
            isLoading={isSubmitting}
          >
            {isSubmitting ? 'Launching Test Run...' : 'Execute Test Run'}
          </Button>
        </>
      }
    >
      <div className="space-y-5 text-xs">
        {validationError && (
          <div className="p-3 bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2 rounded-none" role="alert">
            <span className="material-symbols-outlined text-base text-rose-400">error</span>
            <span>{validationError}</span>
          </div>
        )}
          {/* Execution Mode Dropdown Selector */}
          <div>
            <div className="flex items-center gap-1.5 mb-1.5 font-['Outfit',sans-serif]">
              <label htmlFor="execution-mode-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider">
                Execution Mode
              </label>
              <span
                className="material-symbols-outlined text-sm text-[#4cd7f6] hover:text-[#38c2e0] cursor-help transition-colors"
                title={`Realistic: Simulates standard audience distribution & production path weights.\nFull Coverage: Forces 100% path coverage across all A/B experiment variants & branch splits.`}
                aria-label="Execution mode details"
                data-testid="execution-mode-tooltip-icon"
              >
                help_outline
              </span>
            </div>
            <select
              id="execution-mode-select"
              data-testid="execution-mode-select"
              value={executionMode}
              onChange={(e) => setExecutionMode(e.target.value as 'realistic' | 'forced_variant_coverage')}
              style={{ fontFamily: "'Outfit', sans-serif" }}
              className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
            >
              <option value="realistic" style={{ fontFamily: "'Outfit', sans-serif" }}>Realistic Mode</option>
              <option value="forced_variant_coverage" style={{ fontFamily: "'Outfit', sans-serif" }}>Full Coverage Mode</option>
            </select>
          </div>
          {/* Static List Audience Selector */}
          <div>
            <label htmlFor="static-list-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
              Audience Source (Static List)
            </label>
            <select
              id="static-list-select"
              data-testid="static-list-select"
              value={selectedStaticListId}
              onChange={(e) => setSelectedStaticListId(e.target.value)}
              style={{ fontFamily: "'Outfit', sans-serif" }}
              className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#4cd7f6]/60 text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
            >
              <option value="" style={{ fontFamily: "'Outfit', sans-serif" }}>Select a static audience list...</option>
              {staticLists.length === 0 && (
                <option value="list-static-001" style={{ fontFamily: "'Outfit', sans-serif" }}>Default Audience List (list-static-001)</option>
              )}
              {staticLists.map((list) => (
                <option key={list.list_id} value={list.list_id} style={{ fontFamily: "'Outfit', sans-serif" }}>
                  {list.name || list.list_id} ({list.item_count || 0} members)
                </option>
              ))}
            </select>
          </div>


          {/* Fake Provider Selector */}
          <div>
            <label className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
              Fake Provider Selector (Mock Dispatchers)
            </label>
            <div className="flex gap-3 flex-wrap mt-2 text-xs font-['Outfit',sans-serif]">
              {[
                { id: 'mock-sendgrid', label: 'Mock SendGrid (Email)' },
                { id: 'mock-twilio', label: 'Mock Twilio (SMS)' },
                { id: 'mock-webhook', label: 'Mock Webhook' },
                { id: 'mock-push', label: 'Mock Push Service' },
              ].map((provider) => (
                <label key={provider.id} className="inline-flex items-center gap-2 text-[#dfe2f1] cursor-pointer bg-[#171b26] px-3.5 py-2 rounded-none border border-[#464554] hover:border-[#4cd7f6]/60 transition-all">
                  <input
                    type="checkbox"
                    checked={fakeProviders.includes(provider.id)}
                    onChange={() => toggleProvider(provider.id)}
                    className="accent-[#4cd7f6] rounded-none cursor-pointer"
                  />
                  <span>{provider.label}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Expiry Duration */}
          <div>
            <label htmlFor="test-run-expiry-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
              Test Run Expiry Duration
            </label>
            <select
              id="test-run-expiry-select"
              value={expiryHours}
              onChange={(e) => setExpiryHours(e.target.value)}
              style={{ fontFamily: "'Outfit', sans-serif" }}
              className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
            >
              <option value="1" style={{ fontFamily: "'Outfit', sans-serif" }}>1 Hour</option>
              <option value="24" style={{ fontFamily: "'Outfit', sans-serif" }}>24 Hours (Default)</option>
              <option value="168" style={{ fontFamily: "'Outfit', sans-serif" }}>7 Days</option>
            </select>
          </div>

        </div>
    </Modal>
  );
}
