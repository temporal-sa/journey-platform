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
  fixturePack: string;
  fakeProviders: string[];
  expiryHours: number;
  targetCount: number;
  suppressionCount: number;
  staticListId?: string;
}

export function TestRunModal({
  isOpen,
  onClose,
  onStartTestRun,
  draftId = 'draft-101',
}: TestRunModalProps) {
  const [executionMode, setExecutionMode] = useState<'realistic' | 'forced_variant_coverage'>('realistic');
  const [fixturePack, setFixturePack] = useState('Standard Fixture Pack (50 contacts)');
  const [fakeProviders, setFakeProviders] = useState<string[]>(['mock-sendgrid', 'mock-twilio']);
  const [expiryHours, setExpiryHours] = useState('24');
  const [staticLists, setStaticLists] = useState<StaticList[]>([]);
  const [selectedStaticListId, setSelectedStaticListId] = useState<string>('');
  const [isSubmitting, setIsSubmitting] = useState(false);

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

  // Determine target member count based on selected fixture pack size
  const targetCount = fixturePack.includes('250') ? 250 : fixturePack.includes('10') ? 10 : 50;

  const toggleProvider = (provider: string) => {
    setFakeProviders((prev) =>
      prev.includes(provider) ? prev.filter((p) => p !== provider) : [...prev, provider]
    );
  };

  const handleRunSubmit = async () => {
    setIsSubmitting(true);
    try {
      const payload: Partial<TestRun> = {
        schema_version: '1.0',
        test_run_id: `tr-${Date.now().toString().slice(-6)}`,
        draft_id: draftId,
        static_list_id: selectedStaticListId || undefined,
        mock_inputs: {
          execution_mode: executionMode,
          fixture_pack: fixturePack,
          fake_providers: fakeProviders,
          expiry_hours: parseInt(expiryHours, 10),
        },
      };
      if (onStartTestRun) {
        await onStartTestRun({
          draftId,
          executionMode,
          fixturePack,
          fakeProviders,
          expiryHours: parseInt(expiryHours, 10),
          targetCount,
          suppressionCount: 0,
          staticListId: selectedStaticListId || undefined,
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
          {/* Execution Mode Pill Toggle (matching toggle.jpeg design) */}
          <div className="p-3 bg-[#11141d] border border-[#464554] rounded-none flex items-center justify-between gap-4 font-['Outfit',sans-serif]">
            <div className="flex items-center gap-1.5 min-w-0">
              <label className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider">
                Execution Mode:
              </label>
              <span className="text-xs font-bold text-white truncate">
                {executionMode === 'realistic' ? 'Realistic' : '100% Coverage'}
              </span>
              <span
                title={`Realistic: Simulates standard audience distribution & production path weights.\n100% Coverage: Forces 100% path coverage across all A/B experiment variants & branch splits.`}
                aria-label="Execution mode details"
                data-testid="execution-mode-tooltip-icon"
              >
                info
              </span>
            </div>

            <div className="flex items-center gap-3 shrink-0">
              <button
                type="button"
                onClick={() => setExecutionMode('realistic')}
                data-testid="mode-toggle-realistic"
                style={{ background: 'transparent', border: 'none', outline: 'none' }}
                className={`text-xs transition-colors cursor-pointer bg-transparent border-0 ${
                  executionMode === 'realistic' ? 'text-white font-bold' : 'text-[#908fa0] hover:text-[#dfe2f1]'
                }`}
              >
                Realistic
              </button>

              {/* Glowing Pill Switch matching toggle.jpeg */}
              <button
                type="button"
                role="switch"
                aria-label="Toggle Execution Mode"
                aria-checked={executionMode === 'forced_variant_coverage'}
                data-testid="execution-mode-toggle-switch"
                onClick={() =>
                  setExecutionMode((prev) =>
                    prev === 'realistic' ? 'forced_variant_coverage' : 'realistic'
                  )
                }
                style={{
                  width: '64px',
                  height: '32px',
                  borderRadius: '9999px',
                  padding: '3px',
                  background: executionMode === 'forced_variant_coverage' ? '#0070f3' : '#1e293b',
                  borderColor: executionMode === 'forced_variant_coverage' ? '#38bdf8' : '#475569',
                  borderWidth: '2px',
                  borderStyle: 'solid',
                  boxShadow: executionMode === 'forced_variant_coverage' ? '0 0 16px rgba(0, 112, 243, 0.75)' : 'inset 0 2px 4px rgba(0,0,0,0.5)',
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  transition: 'all 0.3s ease-in-out',
                  outline: 'none',
                }}
                className="relative shrink-0 focus:outline-none"
              >
                <span
                  style={{
                    width: '22px',
                    height: '22px',
                    borderRadius: '50%',
                    backgroundColor: '#ffffff',
                    boxShadow: '0 2px 5px rgba(0, 0, 0, 0.3)',
                    transform: executionMode === 'forced_variant_coverage' ? 'translateX(32px)' : 'translateX(0px)',
                    transition: 'transform 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    display: 'block',
                    pointerEvents: 'none',
                  }}
                />
              </button>

              <button
                type="button"
                onClick={() => setExecutionMode('forced_variant_coverage')}
                data-testid="mode-toggle-coverage"
                style={{ background: 'transparent', border: 'none', outline: 'none' }}
                className={`text-xs transition-colors cursor-pointer bg-transparent border-0 ${
                  executionMode === 'forced_variant_coverage' ? 'text-white font-bold' : 'text-[#908fa0] hover:text-[#dfe2f1]'
                }`}
              >
                100% Coverage
              </button>
            </div>
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
              <option value="" style={{ fontFamily: "'Outfit', sans-serif" }}>Default Mock Fixture Pack (No Static List)</option>
              {staticLists.map((list) => (
                <option key={list.list_id} value={list.list_id} style={{ fontFamily: "'Outfit', sans-serif" }}>
                  {list.name || list.list_id} ({list.item_count || 0} members)
                </option>
              ))}
            </select>
          </div>

          {/* Fixture Pack Picker */}
          <div>
            <label htmlFor="fixture-pack-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
              Fixture Pack Picker (Fallback Mode)
            </label>
            <select
              id="fixture-pack-select"
              value={fixturePack}
              onChange={(e) => setFixturePack(e.target.value)}
              style={{ fontFamily: "'Outfit', sans-serif" }}
              className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
            >
              <option value="Standard Fixture Pack (50 contacts)" style={{ fontFamily: "'Outfit', sans-serif" }}>Standard Fixture Pack (50 contacts)</option>
              <option value="Edge Cases & Invalid Emails (10 contacts)" style={{ fontFamily: "'Outfit', sans-serif" }}>Edge Cases & Invalid Emails (10 contacts)</option>
              <option value="High Volume Stress Pack (250 contacts)" style={{ fontFamily: "'Outfit', sans-serif" }}>High Volume Stress Pack (250 contacts)</option>
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

          {/* Target Count & Execution Configuration Preview */}
          <div className="p-4 rounded-none bg-[#171b26] border border-[#464554] space-y-3 font-['Outfit',sans-serif]">
            <h3 className="font-['Outfit'] font-bold text-xs text-white uppercase tracking-wider">
              Target Count & Execution Configuration
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="p-3 rounded-none bg-[#4cd7f6]/10 border border-[#4cd7f6]/20 text-[#4cd7f6]">
                <div className="text-[10px] text-[#908fa0] uppercase">Target Member Count</div>
                <div className="text-lg font-bold text-white mt-0.5">{targetCount} members</div>
              </div>
              <div className="p-3 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1]">
                <div className="text-[10px] text-[#908fa0] uppercase">Expiry Window</div>
                <div className="text-lg font-bold text-white mt-0.5">{expiryHours} Hours</div>
              </div>
            </div>
            <div className="text-[11px] text-[#908fa0]">
              <strong className="text-[#dfe2f1]">Dispatchers:</strong> {fakeProviders.length > 0 ? fakeProviders.join(', ') : 'None selected'}
            </div>
          </div>
        </div>
    </Modal>
  );
}
