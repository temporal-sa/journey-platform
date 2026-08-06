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
      className="fixed inset-0 bg-[#0B0F19]/85 backdrop-blur-md flex items-center justify-center z-50 p-4 sm:p-6 font-['Outfit',sans-serif]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="agy-context-modal-title"
      data-testid="agy-context-modal"
    >
      <div className="bg-[#0F131D]/95 backdrop-blur-xl border border-[#4cd7f6]/40 rounded-none w-full max-w-3xl max-h-[85vh] flex flex-col shadow-2xl shadow-black/90 overflow-hidden glass-modal shrink-0">
        {/* Header */}
        <div className="bg-[#171b26] p-5 border-b border-[#464554] flex items-center justify-between shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-none bg-[#4cd7f6]/15 border border-[#4cd7f6]/40 flex items-center justify-center text-[#4cd7f6] shrink-0 shadow-[0_0_12px_rgba(76,215,246,0.2)]">
              <span className="material-symbols-outlined text-xl">smart_toy</span>
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 rounded-none bg-[#4cd7f6]/20 text-[#4cd7f6] border border-[#4cd7f6]/40 font-mono text-[10px] font-bold uppercase tracking-wider">
                  ANTIGRAVITY AI ASSISTANT
                </span>
                <h2 id="agy-context-modal-title" className="font-['Outfit'] font-bold text-lg text-white tracking-wide">
                  Export Prompt Context
                </h2>
              </div>
              <p className="text-xs text-[#908fa0] mt-0.5 font-['Outfit']">
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

        {/* Content Body (Scrollable Container) */}
        <div className="p-6 flex-1 min-h-0 flex flex-col space-y-4 overflow-hidden text-xs font-['Outfit',sans-serif]">
          {/* Obsidian Theme Instruction Bar */}
          <div className="p-3 bg-[#171b26] border border-[#464554] text-[#dfe2f1] text-xs font-mono flex items-center justify-between gap-3 shrink-0">
            <div className="flex items-center gap-2 min-w-0">
              <span className="material-symbols-outlined text-[#4cd7f6] text-base shrink-0">info</span>
              <span className="truncate">Copy markdown snippet into Antigravity AI chat for context-aware assistance.</span>
            </div>
            <button
              onClick={handleCopy}
              data-testid="agy-copy-context-btn"
              className={`px-3.5 py-1.5 rounded-none font-mono text-xs font-bold uppercase tracking-wider transition-all border cursor-pointer shrink-0 flex items-center gap-1.5 ${
                copied
                  ? 'bg-[#10b981]/20 text-[#6ee7b7] border-[#10b981]/60 shadow-[0_0_12px_rgba(16,185,129,0.2)]'
                  : 'bg-[#4cd7f6] hover:opacity-90 text-[#400071] border-none shadow-none active:scale-95'
              }`}
              style={copied ? {} : { backgroundColor: '#4cd7f6', color: '#400071' }}
            >
              <span className="material-symbols-outlined text-sm">
                {copied ? 'check' : 'content_copy'}
              </span>
              <span>{copied ? 'COPIED!' : 'COPY TO CLIPBOARD'}</span>
            </button>
          </div>

          {/* Scrollable Preformatted Text Output */}
          <pre className="flex-1 min-h-0 overflow-y-auto overflow-x-auto bg-[#0b0e17] p-4 border border-[#464554] text-[#dfe2f1] font-mono text-[11px] leading-relaxed whitespace-pre-wrap selection:bg-[#4cd7f6]/30 shadow-inner rounded-none">
            {formattedContext}
          </pre>
        </div>

        {/* Footer */}
        <div className="p-4 bg-[#171b26] border-t border-[#464554] flex justify-end items-center gap-3 shrink-0">
          <button
            onClick={handleCopy}
            className={`px-4 py-2 rounded-none font-mono text-xs font-bold uppercase tracking-wider transition-all border cursor-pointer flex items-center gap-2 ${
              copied
                ? 'bg-[#10b981]/20 text-[#6ee7b7] border-[#10b981]/60'
                : 'bg-[#4cd7f6] hover:opacity-90 text-[#400071] border-none shadow-none'
            }`}
            style={copied ? {} : { backgroundColor: '#4cd7f6', color: '#400071' }}
          >
            <span className="material-symbols-outlined text-sm">
              {copied ? 'check' : 'content_copy'}
            </span>
            <span>{copied ? 'COPIED TO CLIPBOARD' : 'COPY CONTEXT'}</span>
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 bg-[#1c1f2a] hover:bg-[#262a35] text-[#dfe2f1] border border-[#464554] text-xs font-mono uppercase tracking-wider font-semibold transition-all cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
