import { useState, ChangeEvent } from 'react';
import type { StaticList } from '../../types/api';
import { JourneyApiClient } from '../../api/client';
import { DegradedStateView } from '../DegradedStateView';
import { Button } from '../common/Button';
import { Modal } from '../common/Modal';
import { Badge } from '../common/Badge';
export interface StaticListUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUploadSuccess?: (list: Partial<StaticList> & { rowCount: number; verifiedContacts: number; formulaRejections: number; expires_at?: string }) => void;
  onUploadError?: (error: string) => void;
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
  onUploadError,
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
      description: selectedVersion,
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
      onUploadError?.(message);
    }
  };

  const getExpiryLabel = () => {
    const hours = parseInt(ttlHours, 10);
    const date = new Date(Date.now() + hours * 3600 * 1000);
    return `${date.toLocaleDateString()} ${date.toLocaleTimeString()} (${hours}h TTL)`;
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Static List CSV Upload"
      icon="upload_file"
      iconAccentColor="#4cd7f6"
      maxWidth="3xl"
      testId="static-list-upload-modal"
      ariaLabelledBy="static-list-upload-title"
      footer={
        <>
          <Button variant="secondary-dark" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary-teal"
            size="lg"
            onClick={handleUploadSubmit}
            isLoading={isProcessing}
          >
            {isProcessing ? 'Uploading & Indexing...' : uploadComplete ? 'Completed!' : 'Confirm Static List Upload'}
          </Button>
        </>
      }
    >
      <div className="space-y-5 text-xs">
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
              <label htmlFor="list-id-input" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
                List ID (Target Identifier)
              </label>
              <input
                id="list-id-input"
                type="text"
                value={listId}
                onChange={(e) => setListId(e.target.value)}
                style={{ fontFamily: "'Outfit', sans-serif" }}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] focus:ring-1 focus:ring-[#4cd7f6] font-['Outfit',sans-serif]"
              />
            </div>
            <div>
              <label htmlFor="list-name-input" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
                List Name
              </label>
              <input
                id="list-name-input"
                type="text"
                value={listName}
                onChange={(e) => setListName(e.target.value)}
                style={{ fontFamily: "'Outfit', sans-serif" }}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] focus:ring-1 focus:ring-[#4cd7f6] font-['Outfit',sans-serif]"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="list-version-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
                Immutable List Version
              </label>
              <select
                id="list-version-select"
                value={selectedVersion}
                onChange={(e) => setSelectedVersion(e.target.value)}
                style={{ fontFamily: "'Outfit', sans-serif" }}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
              >
                <option value="v1 (immutable)" style={{ fontFamily: "'Outfit', sans-serif" }}>v1 - Initial Commit (Immutable)</option>
                <option value="v2 (immutable)" style={{ fontFamily: "'Outfit', sans-serif" }}>v2 - Draft Delta (Immutable)</option>
              </select>
            </div>
            <div>
              <label htmlFor="list-ttl-select" className="block text-[10px] font-semibold text-[#908fa0] uppercase tracking-wider mb-1.5 font-['Outfit',sans-serif]">
                Expiry Duration (TTL)
              </label>
              <select
                id="list-ttl-select"
                value={ttlHours}
                onChange={(e) => setTtlHours(e.target.value)}
                style={{ fontFamily: "'Outfit', sans-serif" }}
                className="w-full px-3.5 py-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1] text-xs focus:outline-none focus:border-[#4cd7f6] font-['Outfit',sans-serif]"
              >
                <option value="1" style={{ fontFamily: "'Outfit', sans-serif" }}>1 Hour</option>
                <option value="24" style={{ fontFamily: "'Outfit', sans-serif" }}>24 Hours (Default)</option>
                <option value="168" style={{ fontFamily: "'Outfit', sans-serif" }}>7 Days</option>
                <option value="720" style={{ fontFamily: "'Outfit', sans-serif" }}>30 Days</option>
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
              className="inline-flex items-center gap-2 px-4 py-2.5 rounded-none bg-[#4cd7f6] hover:bg-[#38c2e0] text-[#003640] font-bold text-xs shadow-lg shadow-[#4cd7f6]/20 transition-all cursor-pointer"
            >
              <span className="material-symbols-outlined text-base">folder_open</span>
              <span>Choose CSV File</span>
            </label>
            {selectedFile ? (
              <p className="mt-2.5 text-xs text-[#4cd7f6]">
                Selected: <strong className="text-white">{selectedFile.name}</strong> ({(selectedFile.size / 1024).toFixed(1)} KB)
              </p>
            ) : (
              <p className="mt-2.5 text-xs text-[#908fa0]">
                Upload CSV file containing headers: <span className="text-[#4cd7f6] font-semibold font-['Outfit',sans-serif]">user_id, email</span>
              </p>
            )}
          </div>

          {/* Upload Progress Bar */}
          {(isProcessing || uploadProgress > 0) && (
            <div className="space-y-1">
              <div className="flex justify-between text-[10px] text-[#908fa0]">
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
            <div className="grid grid-cols-3 gap-3">
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


          {/* Masked Member Preview Table */}
          {previews.length > 0 && (
            <div className="space-y-2">
              <h4 className="font-['Outfit'] font-bold text-xs text-white">
                Masked Member Preview ({previews.length} members)
              </h4>
              <div className="max-h-40 overflow-y-auto rounded-none border border-[#464554] bg-[#11141d]">
                <table className="w-full text-left text-xs font-['Outfit',sans-serif]">
                  <thead>
                    <tr className="bg-[#171b26] border-b border-[#464554] text-[#908fa0] text-[10px] uppercase font-['Outfit',sans-serif]">
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
    </Modal>
  );
}
