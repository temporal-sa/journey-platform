import { useState } from 'react';
import { useEditorStore } from '../../stores/editorStore';

interface AgyContextModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AgyContextModal({ isOpen, onClose }: AgyContextModalProps) {
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const validationResult = useEditorStore((s) => s.validationResult);
  const selectedNodeId = useEditorStore((s) => s.selectedNodeId);

  const [copied, setCopied] = useState(false);

  if (!isOpen) return null;

  const nodeCount = currentDraft?.nodes?.length || 0;
  const edgeCount = currentDraft?.edges?.length || 0;
  const errors = validationResult?.errors || [];
  const warnings = validationResult?.warnings || [];

  const formattedContext = `### Journey Canvas Context Dumper (AGY Prompt Snapshot)

**Journey Name**: ${currentDraft?.name || 'Untitled Journey'}
**Draft ID**: ${currentDraft?.id || 'N/A'} (Revision: v${currentDraft?.version || 1})
**Graph Statistics**: ${nodeCount} Nodes, ${edgeCount} Edges
**Selected Node**: ${selectedNodeId ? `Node ID "${selectedNodeId}"` : 'None'}

#### Topological Validation Status
- **Is Valid**: ${validationResult?.is_valid ? '✅ Valid' : '❌ Has Issues'}
- **Errors (${errors.length})**: ${errors.length > 0 ? errors.map((e) => `\n  - [${e.code}] ${e.message}`).join('') : 'None'}
- **Warnings (${warnings.length})**: ${warnings.length > 0 ? warnings.map((w) => `\n  - [${w.code}] ${w.message}`).join('') : 'None'}

#### Graph Representation (Nodes Summary)
\`\`\`json
${JSON.stringify(
  (currentDraft?.nodes || []).map((n) => ({
    id: n.id,
    type: n.type,
    label: n.data?.label,
    config: n.data?.config,
  })),
  null,
  2
)}
\`\`\`
`;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(formattedContext);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback
    }
  };

  return (
    <div
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center z-50 p-4 font-['Outfit',sans-serif]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="agy-context-modal-title"
      data-testid="agy-context-modal"
    >
      <div className="bg-[#0F131D]/95 backdrop-blur-xl border border-[#4cd7f6]/40 rounded-none w-full max-w-2xl max-h-[85vh] my-auto flex flex-col shadow-2xl shadow-black/80 overflow-hidden glass-modal shrink-0">
        {/* Header */}
        <div className="bg-[#171b26] p-5 border-b border-[#464554] flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-none bg-[#4cd7f6]/15 border border-[#4cd7f6]/40 flex items-center justify-center text-[#4cd7f6] shrink-0">
              <span className="material-symbols-outlined text-xl">smart_toy</span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 rounded-none bg-[#4cd7f6]/20 text-[#4cd7f6] border border-[#4cd7f6]/40 font-mono text-[10px] font-bold uppercase tracking-wider">
                  ANTIGRAVITY AI ASSISTANT
                </span>
                <h2 id="agy-context-modal-title" className="font-['Outfit'] font-bold text-lg text-white">
                  Export Prompt Context
                </h2>
              </div>
              <p className="text-xs text-[#908fa0] mt-0.5">
                Formatted markdown snapshot of current canvas state & validation diagnostic report.
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="w-8 h-8 rounded-none bg-[#1c1f2a] text-[#908fa0] hover:text-white hover:bg-white/10 flex items-center justify-center transition-all border border-[#464554] cursor-pointer shrink-0"
          >
            <span className="material-symbols-outlined text-lg">close</span>
          </button>
        </div>

        {/* Content Body */}
        <div className="p-6 overflow-y-auto flex-1 space-y-4 text-xs font-['Outfit',sans-serif]">
          <div className="p-3 bg-[#4cd7f6]/10 border border-[#4cd7f6]/30 text-[#dfe2f1] text-xs font-mono flex items-center justify-between">
            <span>Copy this snippet into Antigravity chat for AI-assisted debugging or journey iteration.</span>
            <button
              onClick={handleCopy}
              data-testid="agy-copy-context-btn"
              className={`px-3 py-1.5 rounded-none font-mono text-xs font-bold transition-all border cursor-pointer ${
                copied
                  ? 'bg-[#10b981]/20 text-[#10b981] border-[#10b981]/40'
                  : 'bg-[#4cd7f6]/20 text-[#4cd7f6] border-[#4cd7f6]/50 hover:bg-[#4cd7f6]/30'
              }`}
            >
              {copied ? '✓ COPIED!' : 'COPY TO CLIPBOARD'}
            </button>
          </div>

          <pre className="bg-[#11141d] p-4 border border-[#464554] text-[#dfe2f1] font-mono text-[11px] overflow-x-auto max-h-[350px] whitespace-pre-wrap selection:bg-[#4cd7f6]/30">
            {formattedContext}
          </pre>
        </div>

        {/* Footer */}
        <div className="p-4 bg-[#171b26] border-t border-[#464554] flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554] text-xs font-mono uppercase tracking-wider font-semibold transition-all cursor-pointer"
          >
            Done
          </button>
        </div>
      </div>
    </div>
  );
}
