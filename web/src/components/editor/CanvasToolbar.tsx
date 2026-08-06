import React from 'react';

interface CanvasToolbarProps {
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onFitView: () => void;
  onDeleteSelected: () => void;
  hasSelection: boolean;
  isPanMode: boolean;
  onTogglePanMode: () => void;
  isLocked?: boolean;
  onToggleLock?: () => void;
  onSaveDraft?: () => void;
  onSimulateConflict?: () => void;
  onPublish?: () => void;
  onTestMode?: () => void;
  onUploadStaticList?: () => void;
  onLaunchTestRun?: () => void;
  onKeyboardShortcuts?: () => void;
  onAgyContext?: () => void;
  snapToGridEnabled: boolean;
  onToggleSnapToGrid: () => void;
  onSnapAllNodesToGrid: () => void;
  onAutoArrangeHorizontal: () => void;
  onAutoArrangeVertical: () => void;
}

export function CanvasToolbar({
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onZoomIn,
  onZoomOut,
  onFitView,
  onDeleteSelected,
  hasSelection,
  isPanMode,
  onTogglePanMode,
  isLocked = false,
  onToggleLock,
  onSaveDraft,
  onSimulateConflict,
  onPublish,
  onTestMode,
  onUploadStaticList,
  onLaunchTestRun,
  onKeyboardShortcuts,
  onAgyContext,
  snapToGridEnabled,
  onToggleSnapToGrid,
  onSnapAllNodesToGrid,
  onAutoArrangeHorizontal,
  onAutoArrangeVertical,
}: CanvasToolbarProps) {
  const baseBtnStyle: React.CSSProperties = {
    background: 'transparent',
    backgroundColor: 'transparent',
    border: 'none',
    outline: 'none',
    boxShadow: 'none',
    WebkitAppearance: 'none',
  };

  return (
    <div
      className="absolute top-0 left-0 right-0 w-full z-40 flex items-center justify-start gap-1 px-4 py-2 bg-[#0F131D]/40 backdrop-blur-md border-none text-[#DFE2F1] transition-all select-none overflow-x-auto min-w-0 font-['Outfit']"
      data-testid="canvas-toolbar"
    >
      {/* History & Selection Group */}
      <div className="flex items-center gap-1 shrink-0">
        {/* Undo Button */}
        <button
          onClick={onUndo}
          disabled={!canUndo || isLocked}
          style={baseBtnStyle}
          className={`w-10 h-10 p-0 rounded-none flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            canUndo && !isLocked
              ? 'text-[#dfe2f1] hover:text-white hover:bg-white/10'
              : 'text-[#908fa0] cursor-not-allowed'
          }`}
          title="Undo (Ctrl+Z)"
          aria-label="Undo"
          data-testid="toolbar-undo"
        >
          <span className="material-symbols-outlined text-xl">undo</span>
        </button>

        {/* Redo Button */}
        <button
          onClick={onRedo}
          disabled={!canRedo || isLocked}
          style={baseBtnStyle}
          className={`w-10 h-10 p-0 rounded-none flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            canRedo && !isLocked
              ? 'text-[#dfe2f1] hover:text-white hover:bg-white/10'
              : 'text-[#908fa0] cursor-not-allowed'
          }`}
          title="Redo (Ctrl+Y)"
          aria-label="Redo"
          data-testid="toolbar-redo"
        >
          <span className="material-symbols-outlined text-xl">redo</span>
        </button>

        <div className="w-px h-5 bg-[#464554]/50 mx-1" />

        {/* Select / Pan Mode Toggle Button */}
        <button
          onClick={onTogglePanMode}
          style={{
            ...baseBtnStyle,
            backgroundColor: isPanMode ? 'rgba(183, 109, 255, 0.25)' : 'transparent',
          }}
          className={`w-10 h-10 p-0 rounded-none flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            isPanMode
              ? 'flat-icon-btn-active text-[#ddb7ff]'
              : 'text-[#dfe2f1] hover:text-white hover:bg-white/10'
          }`}
          title={isPanMode ? 'Switch to Select Mode' : 'Switch to Pan Mode'}
          aria-label={isPanMode ? 'Switch to Select Mode' : 'Switch to Pan Mode'}
          data-testid="toolbar-pan-mode"
        >
          <span className="material-symbols-outlined text-xl">
            {isPanMode ? 'pan_tool' : 'near_me'}
          </span>
        </button>

        {/* Lock / Interactive Canvas Toggle Button */}
        {onToggleLock && (
          <button
            onClick={onToggleLock}
            style={{
              ...baseBtnStyle,
              backgroundColor: isLocked ? 'rgba(251, 191, 36, 0.2)' : 'transparent',
            }}
            className={`w-10 h-10 p-0 rounded-none flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
              isLocked
                ? 'text-amber-300'
                : 'text-[#dfe2f1] hover:text-white hover:bg-white/10'
            }`}
            title={isLocked ? 'Unlock Canvas (Interactive)' : 'Lock Canvas (Read Only)'}
            aria-label={isLocked ? 'Unlock Canvas' : 'Lock Canvas'}
            data-testid="toolbar-lock-toggle"
          >
            <span className="material-symbols-outlined text-xl">
              {isLocked ? 'lock' : 'lock_open'}
            </span>
          </button>
        )}

        {/* Delete Selected Button */}
        <button
          onClick={onDeleteSelected}
          disabled={!hasSelection || isLocked}
          style={baseBtnStyle}
          className={`w-10 h-10 p-0 rounded-none flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            hasSelection && !isLocked
              ? 'text-rose-400 hover:text-rose-300 hover:bg-rose-500/20'
              : 'text-[#908fa0] cursor-not-allowed'
          }`}
          title="Delete selected nodes/edges (Delete)"
          aria-label="Delete selected"
          data-testid="toolbar-delete"
        >
          <span className="material-symbols-outlined text-xl">delete</span>
        </button>
      </div>

      <div className="w-px h-5 bg-[#464554]/40 mx-1 shrink-0" />

      {/* Action Group: Layout & Canvas Controls */}
      <div className="flex items-center gap-1 shrink-0">
        {/* Toggle Snap to Grid Button */}
        <button
          onClick={onToggleSnapToGrid}
          style={{
            ...baseBtnStyle,
            backgroundColor: snapToGridEnabled ? 'rgba(183, 109, 255, 0.2)' : 'transparent',
          }}
          className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            snapToGridEnabled
              ? 'text-[#ddb7ff]'
              : 'text-[#dfe2f1] hover:text-white hover:bg-white/10'
          }`}
          title={snapToGridEnabled ? 'Disable Snap to Grid' : 'Enable Snap to Grid'}
          aria-label={snapToGridEnabled ? 'Disable Snap to Grid' : 'Enable Snap to Grid'}
          data-testid="toolbar-snap-toggle"
        >
          <span className="material-symbols-outlined text-xl">
            {snapToGridEnabled ? 'grid_on' : 'grid_off'}
          </span>
        </button>

        {/* Auto Arrange Horizontal Button */}
        <button
          onClick={onAutoArrangeHorizontal}
          disabled={isLocked}
          style={baseBtnStyle}
          className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            !isLocked
              ? 'text-[#ddb7ff] hover:text-white hover:bg-[#b76dff]/20'
              : 'text-[#908fa0] cursor-not-allowed'
          }`}
          title="Auto Arrange Nodes (Horizontal)"
          aria-label="Auto Arrange Horizontal"
          data-testid="toolbar-auto-arrange-horizontal"
        >
          <span className="material-symbols-outlined text-xl">account_tree</span>
        </button>

        {/* Auto Arrange Vertical Button */}
        <button
          onClick={onAutoArrangeVertical}
          disabled={isLocked}
          style={baseBtnStyle}
          className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
            !isLocked
              ? 'text-[#ddb7ff] hover:text-white hover:bg-[#b76dff]/20'
              : 'text-[#908fa0] cursor-not-allowed'
          }`}
          title="Auto Arrange Nodes (Vertical)"
          aria-label="Auto Arrange Vertical"
          data-testid="toolbar-auto-arrange-vertical"
        >
          <span className="material-symbols-outlined text-xl" style={{ display: 'inline-block', transform: 'rotate(90deg)' }}>account_tree</span>
        </button>

        <div className="w-px h-5 bg-[#464554]/50 mx-1" />

        {/* Zoom In Button */}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onZoomIn();
          }}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-none text-[#dfe2f1] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Zoom In"
          aria-label="Zoom In"
          data-testid="toolbar-zoom-in"
        >
          <span className="material-symbols-outlined text-xl">zoom_in</span>
        </button>

        {/* Zoom Out Button */}
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onZoomOut();
          }}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-none text-[#dfe2f1] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Zoom Out"
          aria-label="Zoom Out"
          data-testid="toolbar-zoom-out"
        >
          <span className="material-symbols-outlined text-xl">zoom_out</span>
        </button>
      </div>

      <div className="w-px h-5 bg-[#464554]/40 mx-1 shrink-0" />

      {/* Action Group: Persistence, Test & Tools */}
      <div className="flex items-center gap-1 shrink-0">
        {/* Save Draft Action */}
        {onSaveDraft && (
          <button
            onClick={onSaveDraft}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-indigo-300 hover:text-white hover:bg-indigo-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
            title="Save Draft (Cmd+S)"
            aria-label="Save Draft"
            data-testid="toolbar-save-draft"
          >
            <span className="material-symbols-outlined text-xl">save</span>
          </button>
        )}

        {/* Publish Action */}
        {onPublish && (
          <button
            onClick={onPublish}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-emerald-300 hover:text-white hover:bg-emerald-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
            title="Publish Workflow"
            aria-label="Publish"
            data-testid="toolbar-publish"
          >
            <span className="material-symbols-outlined text-xl">rocket_launch</span>
          </button>
        )}

        {/* Upload Static List CSV Action */}
        {(onUploadStaticList || onTestMode) && (
          <button
            onClick={onUploadStaticList || onTestMode}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-cyan-300 hover:text-white hover:bg-cyan-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
            title="Upload Static List CSV"
            aria-label="Upload Static List CSV"
            data-testid="toolbar-upload-static-list"
          >
            <span className="material-symbols-outlined text-xl">cloud_upload</span>
          </button>
        )}

        {/* Configure & Launch Test Execution Action */}
        {onLaunchTestRun && (
          <button
            onClick={onLaunchTestRun}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-[#ddb7ff] hover:text-white hover:bg-[#b76dff]/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
            title="Configure & Launch Test Execution"
            aria-label="Launch Test Execution"
            data-testid="toolbar-launch-test-run"
          >
            <span className="material-symbols-outlined text-xl">play_circle</span>
          </button>
        )}

        {/* Keyboard Shortcuts Help */}
        {onKeyboardShortcuts && (
          <button
            onClick={onKeyboardShortcuts}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-[#c7c4d7] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
            title="Keyboard Shortcuts (?)"
            aria-label="Keyboard Shortcuts"
            data-testid="toolbar-shortcuts"
          >
            <span className="material-symbols-outlined text-xl">keyboard</span>
          </button>
        )}

        {/* Antigravity AI Context Exporter */}
        {onAgyContext && (
          <button
            onClick={onAgyContext}
            style={baseBtnStyle}
            className="w-10 h-10 p-0 rounded-none text-[#4cd7f6] hover:text-white hover:bg-[#4cd7f6]/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn border border-[#4cd7f6]/30"
            title="Export AGY Prompt Context (Cmd+Shift+A)"
            aria-label="Export AGY Prompt Context"
            data-testid="toolbar-agy-context"
          >
            <span className="material-symbols-outlined text-xl">smart_toy</span>
          </button>
        )}
      </div>
    </div>
  );
}
