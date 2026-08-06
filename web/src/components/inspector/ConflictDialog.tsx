import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import type { GraphDraft } from '../../types/api';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';
import { Button } from '../common/Button';

const apiClient = new JourneyApiClient();

export interface ConflictDialogProps {
  isOpen: boolean;
  draftId?: string;
  localDraft: GraphDraft | null;
  serverDraft?: GraphDraft | null;
  localETag?: string | null;
  serverETag?: string | null;
  onKeepLocal: () => void;
  onAcceptServer: () => void;
  onMerge?: () => void;
  onCancel: () => void;
}

function ConflictDialogInner({
  isOpen,
  draftId,
  localDraft,
  serverDraft,
  localETag,
  serverETag,
  onKeepLocal,
  onAcceptServer,
  onMerge,
  onCancel,
}: ConflictDialogProps) {
  const targetDraftId = draftId || localDraft?.draft_id;

  const { data: fetchedServerData } = useQuery({
    queryKey: ['serverDraftConflict', targetDraftId],
    enabled: Boolean(isOpen && !serverDraft && targetDraftId),
    queryFn: () => apiClient.getJourneyDraft(targetDraftId!),
  });

  const effectiveServerDraft = serverDraft || fetchedServerData?.draft || null;
  const effectiveServerETag = serverETag || fetchedServerData?.etag || null;

  if (!isOpen || !localDraft) return null;

  const localNodes = localDraft.nodes || [];
  const serverNodes = effectiveServerDraft?.nodes || [];
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="conflict-dialog-title"
      data-testid="etag-conflict-dialog"
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center z-50 p-4 sm:p-6 overflow-y-auto font-['Outfit',sans-serif]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel();
      }}
    >
      <div className="bg-[#0F131D]/95 backdrop-blur-xl border border-[#464554] rounded-none w-full max-w-4xl max-h-[92vh] sm:max-h-[90vh] my-auto flex flex-col shadow-2xl shadow-black/80 overflow-hidden glass-modal">
        {/* Header */}
        <div className="bg-[#171b26] p-4 sm:p-5 border-b border-[#464554] flex items-center justify-between shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-none bg-amber-500/10 border border-amber-500/30 flex items-center justify-center text-amber-400 shrink-0">
              <span className="material-symbols-outlined text-xl">difference</span>
            </div>
            <div>
              <h2
                id="conflict-dialog-title"
                className="font-['Outfit'] font-bold text-lg text-white"
              >
                ETag Version Conflict Detected (412 Precondition Failed)
              </h2>
              <p className="text-xs text-[#908fa0] mt-0.5">
                The journey draft was modified on the server while you were editing. Review side-by-side revision diffs before resolving.
              </p>
            </div>
          </div>

          <button
            onClick={onCancel}
            aria-label="Close conflict resolution modal"
            data-testid="close-conflict-dialog-btn"
            className="w-8 h-8 rounded-none bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-[#464554] cursor-pointer shrink-0 shadow-sm"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        {/* Comparison Body */}
        <div className="p-4 sm:p-6 overflow-y-auto flex-1 space-y-6 min-h-0">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Local Version Box */}
            <div
              data-testid="local-edits-card"
              className="rounded-none border border-[#c0c1ff]/30 bg-[#c0c1ff]/5 p-5 flex flex-col shadow-inner"
            >
              <div className="text-[10px] font-mono font-bold uppercase tracking-wider text-[#c0c1ff] mb-1">
                Your Local Unsaved Edits
              </div>
              <h3 className="font-['Outfit'] font-bold text-base text-white mb-2">
                {localDraft.name}
              </h3>
              <div className="text-xs text-[#dfe2f1] space-y-1 mb-4 font-mono">
                <div>Version: <span className="text-[#c0c1ff] font-bold">v{localDraft?.version || 1}</span></div>
                <div>Nodes Count: <span className="text-[#c0c1ff] font-bold">{localNodes.length}</span></div>
                <div>Edges Count: <span className="text-[#c0c1ff] font-bold">{localDraft?.edges?.length || 0}</span></div>
                {localETag && (
                  <div>ETag: <span className="text-[#c0c1ff] truncate block">{localETag}</span></div>
                )}
              </div>

              <div className="mt-auto border-t border-[#c0c1ff]/20 pt-3">
                <div className="text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-2">
                  Node Breakdown
                </div>
                <div className="max-h-36 overflow-y-auto space-y-1.5 text-xs font-mono">
                  {localNodes.map((n) => (
                    <div
                      key={n.id}
                      className="p-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1]"
                    >
                      <span className="font-semibold text-[#c0c1ff]">{n.name}</span> ({n.type})
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Server Version Box */}
            <div
              data-testid="server-revision-card"
              className="rounded-none border border-emerald-500/30 bg-emerald-500/5 p-5 flex flex-col shadow-inner"
            >
              <div className="text-[10px] font-mono font-bold uppercase tracking-wider text-emerald-400 mb-1">
                Server Revision (Remote)
              </div>
              <h3 className="font-['Outfit'] font-bold text-base text-white mb-2">
                {effectiveServerDraft?.name}
              </h3>
              <div className="text-xs text-[#dfe2f1] space-y-1 mb-4 font-mono">
                <div>Version: <span className="text-emerald-300 font-bold">v{effectiveServerDraft?.version || 1}</span></div>
                <div>Nodes Count: <span className="text-emerald-300 font-bold">{serverNodes.length}</span></div>
                <div>Edges Count: <span className="text-emerald-300 font-bold">{effectiveServerDraft?.edges?.length || 0}</span></div>
                {effectiveServerETag && (
                  <div>ETag: <span className="text-emerald-400 truncate block">{effectiveServerETag}</span></div>
                )}
              </div>

              <div className="mt-auto border-t border-emerald-500/20 pt-3">
                <div className="text-[10px] font-mono font-semibold text-[#908fa0] uppercase tracking-wider mb-2">
                  Node Breakdown
                </div>
                <div className="max-h-36 overflow-y-auto space-y-1.5 text-xs font-mono">
                  {serverNodes.map((n) => (
                    <div
                      key={n.id}
                      className="p-2.5 rounded-none bg-[#11141d] border border-[#464554] text-[#dfe2f1]"
                    >
                      <span className="font-semibold text-emerald-300">{n.name}</span> ({n.type})
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="p-4 sm:p-5 border-t border-[#464554] bg-[#171b26] sticky bottom-0 z-10 shrink-0 flex flex-wrap items-center justify-between gap-3">
          <Button
            onClick={onCancel}
            data-testid="cancel-conflict-btn"
            variant="secondary-dark"
          >
            Cancel
          </Button>

          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            {onMerge && (
              <Button
                onClick={onMerge}
                data-testid="merge-conflict-btn"
                variant="secondary-dark"
                className="bg-[#b76dff]/20 hover:bg-[#b76dff]/30 text-[#ddb7ff] border-[#ddb7ff]/30"
              >
                Merge Local & Server
              </Button>
            )}

            <Button
              onClick={onAcceptServer}
              data-testid="accept-server-btn"
              variant="secondary-dark"
              className="bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-300 border-emerald-500/40"
            >
              Accept Server Revision
            </Button>

            <Button
              onClick={onKeepLocal}
              data-testid="keep-local-btn"
              variant="primary-lavender"
            >
              Overwrite Server with Local Edits
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function ConflictDialog(props: ConflictDialogProps) {
  return (
    <SafeQueryClientProvider>
      <ConflictDialogInner {...props} />
    </SafeQueryClientProvider>
  );
}
