import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../../api/client';
import type { GraphDraft } from '../../types/api';
import { SafeQueryClientProvider } from '../SafeQueryClientProvider';
import { Button } from '../common/Button';
import { Modal } from '../common/Modal';

const apiClient = new JourneyApiClient();

export interface ConflictDialogProps {
  isOpen: boolean;
  localDraft: GraphDraft | null;
  serverDraft?: GraphDraft | null;
  localETag?: string;
  serverETag?: string;
  onKeepLocal: () => void;
  onAcceptServer: () => void;
  onMerge?: () => void;
  onCancel: () => void;
}

function ConflictDialogInner({
  isOpen,
  localDraft,
  serverDraft,
  localETag,
  serverETag,
  onKeepLocal,
  onAcceptServer,
  onMerge,
  onCancel,
}: ConflictDialogProps) {
  const { data: fetchedServerDraft } = useQuery({
    queryKey: ['draft', 'conflict', localDraft?.id],
    queryFn: async () => {
      if (!localDraft?.id) return null;
      return await apiClient.getJourneyDraft(localDraft.id);
    },
    enabled: isOpen && Boolean(localDraft?.id) && !serverDraft,
  });

  const effectiveServerDraft = serverDraft || fetchedServerDraft || null;
  const effectiveServerETag = serverETag || (effectiveServerDraft ? `W/"rev-${effectiveServerDraft.version}"` : undefined);

  if (!isOpen || !localDraft) return null;

  const localNodes = localDraft.nodes || [];
  const serverNodes = effectiveServerDraft?.nodes || [];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onCancel}
      title="ETag Version Conflict Detected (412 Precondition Failed)"
      subtitle="The journey draft was modified on the server while you were editing. Review side-by-side revision diffs before resolving."
      icon="difference"
      iconAccentColor="#fbbf24"
      maxWidth="4xl"
      testId="conflict-dialog"
      ariaLabelledBy="conflict-dialog-title"
      footer={
        <div className="flex flex-wrap items-center justify-between gap-3 w-full">
          <Button variant="secondary-dark" onClick={onCancel} data-testid="cancel-conflict-btn">
            Cancel & Close
          </Button>

          <div className="flex items-center gap-2">
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
      }
    >
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-xs font-['Outfit',sans-serif]">
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
    </Modal>
  );
}

export function ConflictDialog(props: ConflictDialogProps) {
  return (
    <SafeQueryClientProvider>
      <ConflictDialogInner {...props} />
    </SafeQueryClientProvider>
  );
}
