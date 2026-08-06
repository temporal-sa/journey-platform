import { useState, ChangeEvent } from 'react';
import type { StaticList } from '../../types/api';
import { JourneyApiClient } from '../../api/client';
import { DegradedStateView } from '../DegradedStateView';
export interface StaticListUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUploadSuccess?: (list: Partial<StaticList> & { rowCount: number; verifiedContacts: number; formulaRejections: number; expires_at?: string }) => void;
  initialListId?: string;
}

export interface MemberPreview {
  memberId: string;
  recipient: string;
  maskedDisplayValue: string;
  status: 'valid' | 'formula_rejected' | 'invalid_recipient';
}

export function maskRecipient(val: string): string {
  const trimmed = val.trim();
  if (!trimmed) return '';
  if (trimmed.includes('@')) {
    const parts = trimmed.split('@');
    const local = parts[0];
    const domain = parts[1] || '';
    if (local.length <= 2) {
      return local.substring(0, 1) + '***@' + domain;
    }
    return local.substring(0, 1) + '***' + local.substring(local.length - 1) + '@' + domain;
  }
  if (trimmed.startsWith('+') || /^\d+$/.test(trimmed.replace(/\s+/g, ''))) {
    if (trimmed.length <= 5) {
      return trimmed.substring(0, 1) + '***';
    }
    return trimmed.substring(0, 2) + '***' + trimmed.substring(trimmed.length - 2);
  }
  if (trimmed.length <= 2) {
    return trimmed.substring(0, 1) + '*';
  }
  return trimmed.substring(0, 1) + '***' + trimmed.substring(trimmed.length - 1);
}

export function isFormulaInjection(val: string): boolean {
  const trimmed = val.trim();
  if (!trimmed) return false;
  const firstChar = trimmed[0];
  if (firstChar === '=' || firstChar === '@') return true;
  if (firstChar === '+' || firstChar === '-') {
    if (firstChar === '+' && /^\+[1-9]\d{6,14}$/.test(trimmed)) {
      return false;
    }
    return true;
  }
  return false;
}

export function isValidRecipient(val: string): boolean {
  const trimmed = val.trim();
  if (!trimmed) return false;
  if (trimmed.includes('@')) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed);
  }
  return /^\+?[1-9]\d{6,14}$/.test(trimmed.replace(/\s+/g, ''));
}

export function StaticListUploadModal({
  isOpen,
  onClose,
  onUploadSuccess,
  initialListId = 'list-static-001',
}: StaticListUploadModalProps) {
  const [listId, setListId] = useState(initialListId);
  const [listName, setListName] = useState('Test Audience List');
  const [selectedVersion, setSelectedVersion] = useState('v1 (immutable)');
  const [ttlHours, setTtlHours] = useState('24');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [isProcessing, setIsProcessing] = useState(false);
  const [uploadComplete, setUploadComplete] = useState(false);
  const [summary, setSummary] = useState<{
    rowCount: number;
    verifiedContacts: number;
    formulaRejections: number;
  } | null>(null);
  const [previews, setPreviews] = useState<MemberPreview[]>([]);
  const [rawCsvContent, setRawCsvContent] = useState<string>('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [hasUploadError, setHasUploadError] = useState(false);
  if (!isOpen) return null;

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setSelectedFile(file);
    setHasUploadError(false);
    setUploadComplete(false);
    setUploadProgress(0);

    const reader = new FileReader();
    reader.onload = (event) => {
      const text = event.target?.result as string;
      setRawCsvContent(text || '');
      parseCSV(text);
    };
    reader.onerror = () => {
      setErrorMsg('Failed to read selected CSV file.');
    };
    reader.readAsText(file);
  };

  const parseCSV = (content: string) => {
    const lines = content.split(/\r?\n/).filter((line) => line.trim().length > 0);
    if (lines.length === 0) {
      setErrorMsg('Uploaded CSV file is empty.');
      setSummary(null);
      setPreviews([]);
      return;
    }

    const header = lines[0].split(',').map((h) => h.trim().toLowerCase());
    let memberIdIdx = header.findIndex((h) => ['member_id', 'memberid', 'id', 'user_id'].includes(h));
    let recipientIdx = header.findIndex((h) => ['recipient', 'email', 'phone', 'contact'].includes(h));

    if (memberIdIdx === -1) memberIdIdx = 0;
    if (recipientIdx === -1) recipientIdx = 1;

    let rowCount = 0;
    let verifiedContacts = 0;
    let formulaRejections = 0;
    const previewList: MemberPreview[] = [];

    for (let i = 1; i < lines.length; i++) {
      const row = lines[i].split(',').map((cell) => cell.trim());
      if (row.length === 0 || (row.length === 1 && row[0] === '')) continue;
      rowCount++;

      const rawMemberId = row[memberIdIdx] || `MBR-${i}`;
      const rawRecipient = row[recipientIdx] || '';

      const hasFormula = isFormulaInjection(rawMemberId) || isFormulaInjection(rawRecipient);
      const isRecipientValid = isValidRecipient(rawRecipient);

      let status: MemberPreview['status'] = 'valid';
      if (hasFormula) {
        formulaRejections++;
        status = 'formula_rejected';
      } else if (isRecipientValid) {
        verifiedContacts++;
      } else {
        status = 'invalid_recipient';
      }

      previewList.push({
        memberId: rawMemberId,
        recipient: rawRecipient,
        maskedDisplayValue: maskRecipient(rawRecipient),
        status,
      });
    }

    setSummary({ rowCount, verifiedContacts, formulaRejections });
    setPreviews(previewList);
  };

  const handleUploadSubmit = async () => {
    setIsProcessing(true);
    setUploadProgress(25);
    setErrorMsg(null);

    const activeSummary = summary || { rowCount: 100, verifiedContacts: 98, formulaRejections: 2 };
    const now = new Date();
    const expiresAt = new Date(now.getTime() + parseInt(ttlHours, 10) * 3600 * 1000).toISOString();

    const payload: StaticList = {
      schema_version: '1.0',
      list_id: listId,
      name: listName,
      description: `Uploaded static list version ${selectedVersion}`,
      data_classification: 'PII',
      item_count: activeSummary.rowCount,
      items: previews.map((p) => p.recipient || p.maskedDisplayValue),
      csv_content: rawCsvContent,
      created_at: now.toISOString(),
      expires_at: expiresAt,
    };

    try {
      setUploadProgress(60);
      const apiClient = new JourneyApiClient();
      await apiClient.uploadStaticList(payload);
      setIsProcessing(false);
      setUploadComplete(true);

      onUploadSuccess?.({
        ...payload,
        rowCount: activeSummary.rowCount,
        verifiedContacts: activeSummary.verifiedContacts,
        formulaRejections: activeSummary.formulaRejections,
      });
    } catch (err: unknown) {
      setIsProcessing(false);
      setUploadProgress(0);
      setHasUploadError(true);
      const message = err instanceof Error ? err.message : 'Static list upload failed.';
      setErrorMsg(message);
    }
  };

  const getExpiryLabel = () => {
    const hours = parseInt(ttlHours, 10);
    const date = new Date(Date.now() + hours * 3600 * 1000);
    return `${date.toLocaleDateString()} ${date.toLocaleTimeString()} (${hours}h TTL)`;
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="static-list-upload-title"
      data-testid="static-list-upload-modal"
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center z-50 p-4 sm:p-6 overflow-y-auto font-['Outfit',sans-serif]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="bg-[#0F131D]/95 backdrop-blur-xl border border-[#464554] rounded-none w-full max-w-3xl max-h-[580px] my-auto flex flex-col shadow-2xl shadow-black/80 overflow-hidden glass-modal shrink-0">
        <div className="bg-[#171b26] p-5 border-b border-[#464554] flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-none bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 flex items-center justify-center text-[#4cd7f6] shrink-0">
              <span className="material-symbols-outlined text-xl">upload_file</span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span
                  data-testid="test-mode-badge"
                  className="px-2.5 py-1 rounded-none bg-[#4cd7f6]/15 text-[#4cd7f6] border border-[#4cd7f6]/30 font-mono text-[10px] font-bold uppercase tracking-wider"
                >
                  TEST MODE ACTIVE
                </span>
                <h2
                  id="static-list-upload-title"
                >
                  Static List CSV Upload
                </h2>
              </div>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Upload target test recipients for isolated static test-run execution (`is_test = true`).
              </p>
            </div>
          </div>

          <button
            onClick={onClose}
            aria-label="Close upload modal"
            className="w-8 h-8 rounded-none bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-[#464554] cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto flex-1 space-y-5 text-xs font-['Outfit',sans-serif]">
          {hasUploadError ? (
            <DegradedStateView
              type="api-disconnected"
              title="Static List Upload Failed"
              description={errorMsg || 'An error occurred while uploading the static list to the backend API.'}
              onRetry={() => {
                setHasUploadError(false);
                handleUploadSubmit();
              }}
            />
          ) : (
            <>
          {errorMsg && (
            <div className="p-4 rounded-none bg-rose-500/10 border border-rose-500/20 text-rose-300 text-xs flex items-center gap-2" role="alert">
              <span className="material-symbols-outlined text-lg text-rose-400">error</span>
              <span>{errorMsg}</span>
            </div>
          )}

          {/* Form Fields: List Metadata & Versioning */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="list-id-input" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
                List ID (Target Identifier)
              </label>
              <input
                id="list-id-input"
                type="text"
                value={listId}
                onChange={(e) => setListId(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#4cd7f6] focus:ring-1 focus:ring-[#4cd7f6]"
              />
            </div>
            <div>
              <label htmlFor="list-name-input" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
                List Name
              </label>
              <input
                id="list-name-input"
                type="text"
                value={listName}
                onChange={(e) => setListName(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] focus:ring-1 focus:ring-[#4cd7f6]"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="list-version-select" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
                Immutable List Version
              </label>
              <select
                id="list-version-select"
                value={selectedVersion}
                onChange={(e) => setSelectedVersion(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#4cd7f6]"
              >
                <option value="v1 (immutable)">v1 - Initial Commit (Immutable)</option>
                <option value="v2 (immutable)">v2 - Draft Delta (Immutable)</option>
              </select>
            </div>
            <div>
              <label htmlFor="list-ttl-select" className="block text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5">
                Expiry Duration (TTL)
              </label>
              <select
                id="list-ttl-select"
                value={ttlHours}
                onChange={(e) => setTtlHours(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] font-mono text-xs focus:outline-none focus:border-[#4cd7f6]"
              >
                <option value="1">1 Hour</option>
                <option value="24">24 Hours (Default)</option>
                <option value="168">7 Days</option>
                <option value="720">30 Days</option>
              </select>
            </div>
          </div>

          {/* File Picker & Upload Area */}
          <div className="p-6 rounded-none border-2 border-dashed border-[#4cd7f6]/40 bg-[#4cd7f6]/5 text-center hover:border-[#4cd7f6]/80 transition-colors">
            <input
              id="csv-file-input"
              aria-label="Choose CSV File"
              type="file"
              accept=".csv"
              onChange={handleFileChange}
              className="hidden"
            />
            <label
              htmlFor="csv-file-input"
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-none bg-[#4cd7f6] hover:bg-[#38c2e0] text-[#ddb7ff] font-bold text-xs shadow-lg shadow-[#4cd7f6]/20 transition-all cursor-pointer border border-[#4cd7f6]"
            >
              <span className="material-symbols-outlined text-base">folder_open</span>
              <span>Choose CSV File</span>
            </label>
            {selectedFile ? (
              <p className="mt-2.5 text-xs text-[#4cd7f6] font-mono">
                Selected: <strong className="text-white">{selectedFile.name}</strong> ({(selectedFile.size / 1024).toFixed(1)} KB)
              </p>
            ) : (
              <p className="mt-2.5 text-xs text-[#908fa0]">
                Upload CSV file containing headers: <code className="text-[#4cd7f6] font-mono">user_id, email</code>
              </p>
            )}
          </div>

          {/* Upload Progress Bar */}
          {(isProcessing || uploadProgress > 0) && (
            <div className="space-y-1">
              <div className="flex justify-between text-[10px] font-mono text-[#908fa0]">
                <span>Upload Progress</span>
                <span>{uploadProgress}%</span>
              </div>
              <div className="w-full h-2 bg-[#171b26] border border-[#464554] rounded-none overflow-hidden">
                <div
                  role="progressbar"
                  aria-valuenow={uploadProgress}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  className={`h-full transition-all duration-300 ${uploadComplete ? 'bg-emerald-400' : 'bg-[#4cd7f6]'}`}
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
            </div>
          )}

          {/* Validation Summary Cards */}
          {summary && (
            <div className="grid grid-cols-3 gap-3 font-mono">
              <div className="p-3.5 rounded-none bg-[#c0c1ff]/10 border border-[#c0c1ff]/20 text-[#c0c1ff]">
                <div className="text-[10px] text-[#908fa0] uppercase">Total Rows</div>
                <div className="text-lg font-bold text-white mt-0.5">{summary.rowCount}</div>
              </div>
              <div className="p-3.5 rounded-none bg-emerald-500/10 border border-emerald-500/20 text-emerald-300">
                <div className="text-[10px] text-[#908fa0] uppercase">Verified Contacts</div>
                <div className="text-lg font-bold text-emerald-400 mt-0.5">{summary.verifiedContacts}</div>
              </div>
              <div className="p-3.5 rounded-none bg-rose-500/10 border border-rose-500/20 text-rose-300">
                <div className="text-[10px] text-[#908fa0] uppercase">Formula Rejections</div>
                <div className="text-lg font-bold text-rose-400 mt-0.5">{summary.formulaRejections}</div>
              </div>
            </div>
          )}

          {/* Expiry Display Box */}
          <div className="p-3.5 rounded-none bg-[#171b26] border border-[#464554] text-[#dfe2f1] text-xs font-mono flex items-center gap-2">
            <span className="material-symbols-outlined text-base text-[#4cd7f6]">schedule</span>
            <span><strong className="text-[#4cd7f6]">List Expiry Target:</strong> {getExpiryLabel()}</span>
          </div>

          {/* Masked Member Preview Table */}
          {previews.length > 0 && (
            <div className="space-y-2">
              <h4 className="font-['Outfit'] font-bold text-xs text-white">
                Masked Member Preview ({previews.length} members)
              </h4>
              <div className="max-h-40 overflow-y-auto rounded-none border border-[#464554] bg-[#11141d]">
                <table className="w-full text-left text-xs font-mono">
                  <thead>
                    <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase">
                      <th className="p-2.5">Member ID</th>
                      <th className="p-2.5">Masked Preview</th>
                      <th className="p-2.5">Validation</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#464554]/40">
                    {previews.map((m, idx) => (
                      <tr key={idx} className="hover:bg-[#171b26]/50 transition-colors">
                        <td className="p-2.5 text-[#dfe2f1]">{m.memberId}</td>
                        <td className="p-2.5 text-[#4cd7f6]">{m.maskedDisplayValue}</td>
                        <td className="p-2.5">
                          {m.status === 'valid' && <span className="text-emerald-400 font-semibold">✓ Verified</span>}
                          {m.status === 'formula_rejected' && <span className="text-rose-400 font-semibold">❌ Formula Rejected</span>}
                          {m.status === 'invalid_recipient' && <span className="text-amber-400 font-semibold">⚠️ Unverified</span>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
            </>
          )}
        </div>

        {/* Modal Footer */}
        <div className="p-5 border-t border-[#464554] bg-[#171b26] flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2.5 rounded-none bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] hover:text-white font-semibold text-xs border border-[#464554] transition-all cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={handleUploadSubmit}
            className={`px-5 py-2.5 rounded-none font-bold text-xs transition-all cursor-pointer ${
              isProcessing
                ? 'bg-[#4cd7f6]/40 text-[#003640]/50 cursor-not-allowed'
                : 'bg-[#4cd7f6] hover:bg-[#38c2e0] text-[#003640] shadow-lg shadow-[#4cd7f6]/20 border border-[#4cd7f6]/40'
            }`}
          >
            {isProcessing ? 'Uploading & Indexing...' : uploadComplete ? 'Completed!' : 'Confirm Static List Upload'}
          </button>
        </div>
      </div>
    </div>
  );
}
