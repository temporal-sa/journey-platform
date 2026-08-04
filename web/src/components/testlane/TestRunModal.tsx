import { useState } from 'react';
import { JourneyApiClient } from '../../api/client';
import type { TestRun } from '../../types/api';

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
  const [isSubmitting, setIsSubmitting] = useState(false);

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
        mock_inputs: {
          execution_mode: executionMode,
          fixture_pack: fixturePack,
          fake_providers: fakeProviders,
          expiry_hours: parseInt(expiryHours, 10),
        },
      };
      const res = await apiClient.startTestRun(payload);
      const resRecord = (res && typeof res === 'object' ? res : {}) as Record<string, unknown>;
      onStartTestRun?.({
        draftId,
        executionMode,
        fixturePack,
        fakeProviders,
        expiryHours: parseInt(expiryHours, 10),
        targetCount,
        suppressionCount: typeof resRecord.suppressed_members_count === 'number' ? resRecord.suppressed_members_count : 0,
      });
      onClose();
    } catch (err) {
      console.error('Failed to launch test run:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-labelledby="test-run-modal-title"
      aria-modal="true"
      data-testid="test-run-modal"
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center z-50 p-4 font-['Outfit',sans-serif]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-[#0F131D]/95 backdrop-blur-xl border border-[#464554] rounded-2xl w-full max-w-2xl max-h-[580px] my-auto flex flex-col shadow-2xl shadow-black/80 overflow-hidden glass-modal shrink-0">
        {/* Modal Header with Test Mode Badging */}
        <div className="bg-[#171b26] p-5 border-b border-[#464554] flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 flex items-center justify-center text-[#4cd7f6] shrink-0">
              <span className="material-symbols-outlined text-xl">play_circle</span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span
                  data-testid="test-mode-badge"
                  className="px-2 py-0.5 rounded bg-[#4cd7f6]/15 text-[#4cd7f6] border border-[#4cd7f6]/30 font-mono text-[10px] font-bold uppercase tracking-wider"
                >
                  TEST MODE
                </span>
                <h2
                  id="test-run-modal-title"
                  className="font-['Outfit'] font-bold text-lg text-white"
                >
                  Configure Test Run Execution
                </h2>
              </div>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Run Candidate Journey IR against isolated test audience fixtures.
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="w-8 h-8 rounded-xl bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-[#464554] cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>
        {/* Modal Content */}
        <div className="p-6 overflow-y-auto flex-1 space-y-5 text-xs font-['Outfit',sans-serif]">
          {/* Target Journey Draft Header Info */}
          <div className="p-3.5 rounded-xl bg-[#4cd7f6]/10 border border-[#4cd7f6]/20 text-[#4cd7f6] text-xs font-mono flex items-center gap-2">
            <span className="material-symbols-outlined text-base">info</span>
            <span>Target Journey Draft: <strong className="text-white">{draftId}</strong> (Isolated Sandbox Mode)</span>
          </div>
          {/* Execution Mode Selector */}
          <div>
            <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-2">
              Execution Mode
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div
                onClick={() => setExecutionMode('realistic')}
                className={`p-4 rounded-xl border transition-all cursor-pointer ${
                  executionMode === 'realistic'
                    ? 'border-[#4cd7f6] bg-[#4cd7f6]/10 shadow-[0_0_20px_rgba(76,215,246,0.2)]'
                    : 'border-[#464554] bg-[#171b26] hover:border-[#908fa0]'
                }`}
              >
                <div className="font-semibold text-white font-['Outfit'] text-sm">Realistic Mode</div>
                <div className="text-xs text-[#908fa0] mt-1">
                  Simulates standard audience distribution & production path weights.
                </div>
              </div>
              <div
                onClick={() => setExecutionMode('forced_variant_coverage')}
                className={`p-4 rounded-xl border transition-all cursor-pointer ${
                  executionMode === 'forced_variant_coverage'
                    ? 'border-[#4cd7f6] bg-[#4cd7f6]/10 shadow-[0_0_20px_rgba(76,215,246,0.2)]'
                    : 'border-[#464554] bg-[#171b26] hover:border-[#908fa0]'
                }`}
              >
                <div className="font-semibold text-white font-['Outfit'] text-sm">Forced Variant Coverage</div>
                <div className="text-xs text-[#908fa0] mt-1">
                  Forces 100% path coverage across all A/B experiment variants & branch splits.
                </div>
              </div>
            </div>
          </div>

          {/* Fixture Pack Picker */}
          <div>
            <label htmlFor="fixture-pack-select" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
              Fixture Pack Picker
            </label>
            <select
              id="fixture-pack-select"
              value={fixturePack}
              onChange={(e) => setFixturePack(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#11141d] border border-[#464554] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#4cd7f6]"
            >
              <option value="Standard Fixture Pack (50 contacts)">Standard Fixture Pack (50 contacts)</option>
              <option value="Edge Cases & Invalid Emails (10 contacts)">Edge Cases & Invalid Emails (10 contacts)</option>
              <option value="High Volume Stress Pack (250 contacts)">High Volume Stress Pack (250 contacts)</option>
            </select>
          </div>

          {/* Fake Provider Selector */}
          <div>
            <label className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
              Fake Provider Selector (Mock Dispatchers)
            </label>
            <div className="flex gap-3 flex-wrap mt-2 font-mono text-xs">
              {[
                { id: 'mock-sendgrid', label: 'Mock SendGrid (Email)' },
                { id: 'mock-twilio', label: 'Mock Twilio (SMS)' },
                { id: 'mock-webhook', label: 'Mock Webhook' },
                { id: 'mock-push', label: 'Mock Push Service' },
              ].map((provider) => (
                <label key={provider.id} className="inline-flex items-center gap-2 text-[#dfe2f1] cursor-pointer bg-[#171b26] px-3.5 py-2 rounded-xl border border-[#464554] hover:border-[#4cd7f6]/60 transition-all">
                  <input
                    type="checkbox"
                    checked={fakeProviders.includes(provider.id)}
                    onChange={() => toggleProvider(provider.id)}
                    className="accent-[#4cd7f6] rounded cursor-pointer"
                  />
                  <span>{provider.label}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Expiry Duration */}
          <div>
            <label htmlFor="test-run-expiry-select" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
              Test Run Expiry Duration
            </label>
            <select
              id="test-run-expiry-select"
              value={expiryHours}
              onChange={(e) => setExpiryHours(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl bg-[#11141d] border border-[#464554] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#4cd7f6]"
            >
              <option value="1">1 Hour</option>
              <option value="24">24 Hours (Default)</option>
              <option value="168">7 Days</option>
            </select>
          </div>

          {/* Target Count & Execution Configuration Preview */}
          <div className="p-4 rounded-2xl bg-[#171b26] border border-[#464554] space-y-3 font-mono">
            <h3 className="font-['Outfit'] font-bold text-xs text-white uppercase tracking-wider">
              Target Count & Execution Configuration
            </h3>
            <div className="grid grid-cols-2 gap-3">
              <div className="p-3 rounded-xl bg-[#4cd7f6]/10 border border-[#4cd7f6]/20 text-[#4cd7f6]">
                <div className="text-[10px] text-[#908fa0] uppercase">Target Member Count</div>
                <div className="text-lg font-bold text-white mt-0.5">{targetCount} members</div>
              </div>
              <div className="p-3 rounded-xl bg-[#11141d] border border-[#464554] text-[#dfe2f1]">
                <div className="text-[10px] text-[#908fa0] uppercase">Expiry Window</div>
                <div className="text-lg font-bold text-white mt-0.5">{expiryHours} Hours</div>
              </div>
            </div>
            <div className="text-[11px] text-[#908fa0]">
              <strong className="text-[#dfe2f1]">Dispatchers:</strong> {fakeProviders.length > 0 ? fakeProviders.join(', ') : 'None selected'}
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="p-5 border-t border-[#464554] bg-[#171b26] flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2.5 rounded-xl bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] hover:text-white font-semibold text-xs border border-[#464554] transition-all cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={handleRunSubmit}
            data-testid="start-test-run-btn"
            disabled={isSubmitting}
            className={`px-5 py-2.5 rounded-xl font-bold text-xs transition-all cursor-pointer ${
              isSubmitting
                ? 'bg-[#4cd7f6]/40 text-[#003640]/50 cursor-not-allowed'
                : 'bg-[#4cd7f6] hover:bg-[#38c2e0] text-[#003640] shadow-lg shadow-[#4cd7f6]/20 border border-[#4cd7f6]/40'
            }`}
          >
            {isSubmitting ? 'Launching Test Run...' : 'Execute Test Run'}
          </button>
        </div>
      </div>
    </div>
  );
}
