import { useState } from 'react';
import { useEditorStore } from '../../stores/editorStore';
import { Button } from '../common/Button';
import { Modal } from '../common/Modal';

interface AgyContextModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AgyContextModal({ isOpen, onClose }: AgyContextModalProps) {
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const validationResult = useEditorStore((s) => s.validationResult);
  const selectedNodeId = useEditorStore((s) => s.selectedNodeId);

  const [copied, setCopied] = useState(false);

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
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Export Prompt Context"
      icon="smart_toy"
      iconAccentColor="#4cd7f6"
      maxWidth="3xl"
      testId="agy-context-modal"
      ariaLabelledBy="agy-context-modal-title"
      footer={
        <Button
          onClick={handleCopy}
          data-testid="agy-copy-context-btn"
          variant={copied ? 'secondary-dark' : 'primary-cyan'}
          icon={copied ? 'check' : 'content_copy'}
          className={copied ? 'bg-[#10b981]/20 text-[#6ee7b7] border-[#10b981]/60' : ''}
        >
          {copied ? 'Copied!' : 'Copy to clipboard'}
        </Button>
      }
    >
      <pre className="flex-1 min-h-0 overflow-y-auto overflow-x-auto bg-[#0b0e17] p-4 border border-[#464554] text-[#dfe2f1] font-mono text-[11px] leading-relaxed whitespace-pre-wrap selection:bg-[#4cd7f6]/30 shadow-inner rounded-none">
        {formattedContext}
      </pre>
    </Modal>
  );
}
