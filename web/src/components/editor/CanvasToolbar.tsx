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
  onKeyboardShortcuts?: () => void;
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
  onKeyboardShortcuts,
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
      className="absolute top-4 left-4 z-40 flex items-center gap-2 p-2 rounded-full bg-[#0F131D]/80 backdrop-blur-xl border-0 outline-none transition-all"
      data-testid="canvas-toolbar"
    >
      {/* Undo Button */}
      <button
        onClick={onUndo}
        disabled={!canUndo || isLocked}
        style={baseBtnStyle}
        className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
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
        className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
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

      <div className="w-px h-5 bg-[#464554]/50 mx-0.5" />

      {/* Select / Pan Mode Toggle Button */}
      <button
        onClick={onTogglePanMode}
        style={{
          ...baseBtnStyle,
          backgroundColor: isPanMode ? 'rgba(183, 109, 255, 0.25)' : 'transparent',
        }}
        className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
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
          className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
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
        className={`w-10 h-10 p-0 rounded-full flex items-center justify-center transition-all cursor-pointer flat-icon-btn ${
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

      <div className="w-px h-5 bg-[#464554]/50 mx-0.5" />

      {/* Zoom In Button */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onZoomIn();
        }}
        style={baseBtnStyle}
        className="w-10 h-10 p-0 rounded-full text-[#dfe2f1] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
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
        className="w-10 h-10 p-0 rounded-full text-[#dfe2f1] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
        title="Zoom Out"
        aria-label="Zoom Out"
        data-testid="toolbar-zoom-out"
      >
        <span className="material-symbols-outlined text-xl">zoom_out</span>
      </button>

      {/* Fit View Button */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onFitView();
        }}
        style={baseBtnStyle}
        className="w-10 h-10 p-0 rounded-full text-[#ddb7ff] hover:text-[#ddb7ff] hover:bg-[#b76dff]/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
        title="Fit View"
        aria-label="Fit View"
        data-testid="toolbar-fit-view"
      >
        <span className="material-symbols-outlined text-xl">center_focus_strong</span>
      </button>

      <div className="w-px h-5 bg-[#464554]/50 mx-0.5" />

      {/* Save Draft Action */}
      {onSaveDraft && (
        <button
          onClick={onSaveDraft}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-full text-indigo-300 hover:text-white hover:bg-indigo-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Save Draft (Cmd+S)"
          aria-label="Save Draft"
          data-testid="toolbar-save-draft"
        >
          <span className="material-symbols-outlined text-xl">save</span>
        </button>
      )}

      {/* Simulate Conflict Action */}
      {onSimulateConflict && (
        <button
          onClick={onSimulateConflict}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-full text-purple-300 hover:text-white hover:bg-purple-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Simulate ETag Conflict"
          aria-label="Simulate Conflict"
          data-testid="toolbar-sim-conflict"
        >
          <span className="material-symbols-outlined text-xl">difference</span>
        </button>
      )}

      {/* Publish Action */}
      {onPublish && (
        <button
          onClick={onPublish}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-full text-emerald-300 hover:text-white hover:bg-emerald-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Publish Workflow"
          aria-label="Publish"
          data-testid="toolbar-publish"
        >
          <span className="material-symbols-outlined text-xl">rocket_launch</span>
        </button>
      )}
      {/* Test Mode & Static List CSV Upload Action */}
      {onTestMode && (
        <button
          onClick={onTestMode}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-full text-cyan-300 hover:text-white hover:bg-cyan-500/20 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Test Mode & Static List CSV Upload"
          aria-label="Test Mode CSV Upload"
          data-testid="toolbar-test-mode"
        >
          <span className="material-symbols-outlined text-xl">upload_file</span>
        </button>
      )}

      {/* Keyboard Shortcuts Help */}
      {onKeyboardShortcuts && (
        <button
          onClick={onKeyboardShortcuts}
          style={baseBtnStyle}
          className="w-10 h-10 p-0 rounded-full text-[#c7c4d7] hover:text-white hover:bg-white/10 flex items-center justify-center transition-colors cursor-pointer flat-icon-btn"
          title="Keyboard Shortcuts (?)"
          aria-label="Keyboard Shortcuts"
          data-testid="toolbar-shortcuts"
        >
          <span className="material-symbols-outlined text-xl">keyboard</span>
        </button>
      )}
    </div>
  );
}
