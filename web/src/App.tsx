import { useState, useEffect, useRef } from 'react';
import { QueryClient, QueryClientProvider, useQueryClient } from '@tanstack/react-query';
import { ErrorBoundary } from './components/ErrorBoundary';
import { useEditorStore } from './stores/editorStore';
import {
  JourneysPage,
  CatalogPage,
  VersionHistoryPage,
  RunListPage,
  RunDetailPage,
  StaticListsPage,
} from './pages';
import { Canvas } from './components/editor/Canvas';
import { AgyContextModal } from './components/editor/AgyContextModal';
import { NodeInspector, ConflictDialog } from './components/inspector';
import { KeyboardShortcutsModal } from './components/KeyboardShortcutsModal';
import { StaticListUploadModal, TestRunModal } from './components/testlane';
import { DesktopOnlyNotice } from './components/DesktopOnlyNotice';
import { DeveloperPanel } from './components/dev/DeveloperPanel';
import { ParameterModal } from './components/inspector/ParameterModal';
import { Button } from './components/common/Button';
import { Modal } from './components/common/Modal';
import { Toast, ToastMessageType } from './components/common/Toast';
import { BUILD_TAG } from './buildTag';
import { useAnnouncer, LiveAnnouncer } from './hooks/useAnnouncer';
import { ExperimentReportView } from './components/experiments/ExperimentReportView';
import { JourneyApiClient, APIError } from './api/client';
import { isSimulatedApiFailureEnabled } from './api/simulatedFailure';
import type { GraphDraft, GraphNode } from './types/api';
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
  },
});
export type NavigationRoute = 'journeys' | 'catalog' | 'history' | 'runs' | 'run-detail' | 'canvas' | 'experiments' | 'static-lists';

export function getRouteFromHash(): NavigationRoute {
  const hash = window.location.hash.replace(/^#\/?/, '').split('/')[0].split('?')[0];
  const validRoutes: NavigationRoute[] = [
    'journeys',
    'catalog',
    'history',
    'runs',
    'run-detail',
    'canvas',
    'experiments',
    'static-lists',
  ];
  return validRoutes.includes(hash as NavigationRoute) ? (hash as NavigationRoute) : 'journeys';
}

export function InlineJourneyName({
  onRenameSuccess,
  onRenameError,
}: {
  onRenameSuccess?: (newName: string) => void;
  onRenameError?: (msg: string) => void;
}) {
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const setDraftName = useEditorStore((s) => s.setDraftName);
  const markSaved = useEditorStore((s) => s.markSaved);
  const hasUnsavedChanges = useEditorStore((s) => s.hasUnsavedChanges);
  const { announce } = useAnnouncer();

  const [isEditing, setIsEditing] = useState(false);
  const [editedName, setEditedName] = useState(currentDraft?.name || 'Onboarding Flow');

  useEffect(() => {
    if (currentDraft?.name) {
      setEditedName(currentDraft.name);
    }
  }, [currentDraft?.name]);

  const handleSave = async () => {
    setIsEditing(false);
    const trimmed = editedName.trim();
    if (trimmed && trimmed !== currentDraft?.name) {
      setEditedName(trimmed);
      setDraftName(trimmed);
      announce(`Journey name updated to ${trimmed}`, 'polite');

      try {
        const client = new JourneyApiClient();
        const tenantId = currentDraft?.tenant_id || 'default';
        const draftId = currentDraft?.draft_id || 'draft-101';

        const updatedDraft: GraphDraft = currentDraft
          ? { ...currentDraft, name: trimmed }
          : {
              schema_version: '1.0',
              draft_id: draftId,
              tenant_id: tenantId,
              name: trimmed,
              version: 1,
              nodes: [{ id: 'node-start', type: 'trigger', name: 'Start Event' }],
              edges: [],
            };

        const reqHeaders: Record<string, string> = { 'X-Tenant-ID': tenantId };
        if (currentDraft?.content_hash && currentDraft.content_hash !== 'undefined' && currentDraft.content_hash !== 'null') {
          reqHeaders['If-Match'] = currentDraft.content_hash;
        }

        await client.updateJourneyDraft(draftId, updatedDraft, reqHeaders);
        markSaved();
        onRenameSuccess?.(trimmed);
      } catch (err: unknown) {
        if (isSimulatedApiFailureEnabled()) {
          const errMsg = err instanceof Error ? err.message : 'Failed to rename journey.';
          onRenameError?.(errMsg);
          announce(`Failed to update journey name: ${errMsg}`, 'assertive');
        } else {
          onRenameSuccess?.(trimmed);
        }
      }
    } else {
      setEditedName(currentDraft?.name || 'Onboarding Flow');
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      setEditedName(currentDraft?.name || 'Onboarding Flow');
      setIsEditing(false);
    }
  };

  if (isEditing) {
    return (
      <div className="inline-flex items-center gap-2">
        <input
          type="text"
          value={editedName}
          onChange={(e) => setEditedName(e.target.value)}
          onBlur={handleSave}
          onKeyDown={handleKeyDown}
          autoFocus
          data-testid="inline-journey-name-input"
          style={{
            background: '#11141d',
            backgroundColor: '#11141d',
            border: '1px solid #464554',
            outline: 'none',
            boxShadow: 'none',
            WebkitAppearance: 'none',
          }}
          className="rounded-none px-3 py-1 text-sm font-bold text-white font-['Outfit'] min-w-[220px]"
        />
      </div>
    );
  }

  return (
    <div className="inline-flex items-center gap-2">
      <button
        onClick={() => setIsEditing(true)}
        data-testid="inline-journey-name-button"
        title="Click to edit journey name inline"
        aria-label={`Journey name: ${currentDraft?.name || 'Onboarding Flow'}. Click to edit name.`}
        className="px-3 py-1 rounded-none bg-[#c0c1ff]/10 hover:bg-[#c0c1ff]/20 text-[#c0c1ff] hover:text-white border border-[#c0c1ff]/30 hover:border-[#c0c1ff]/60 font-['Outfit'] font-bold text-sm transition-all cursor-pointer inline-flex items-center gap-2 group shadow-sm backdrop-blur-md"
      >
        <span>{currentDraft?.name || 'Onboarding Flow'}</span>
        <span className="material-symbols-outlined text-xs text-[#c0c1ff] opacity-60 group-hover:opacity-100 transition-opacity">
          edit
        </span>
      </button>
      {hasUnsavedChanges && (
        <span
          title="Unsaved changes in journey graph"
          className="px-2 py-0.5 rounded-none bg-amber-500/10 text-amber-400 border border-amber-500/20 font-mono text-[9px] font-bold uppercase tracking-wider"
        >
          UNSAVED
        </span>
      )}
    </div>
  );
}
export function DashboardContent() {
  const {
    selectedNodeId,
    selectedEdgeId,
    isSidebarOpen,
    isInspectorOpen,
    toggleInspector,
    currentDraft,
    setDraft,
    markSaved,
    activeModal,
    openModal,
    closeModal,
    undo,
    redo,
    deleteElements,
  } = useEditorStore();
  const queryClient = useQueryClient();
  const [activeRoute, setActiveRoute] = useState<NavigationRoute>(getRouteFromHash);

  useEffect(() => {
    const currentHashRoute = getRouteFromHash();
    if (currentHashRoute !== activeRoute || window.location.hash !== `#/${activeRoute}`) {
      window.location.hash = `#/${activeRoute}`;
    }
  }, [activeRoute]);

  useEffect(() => {
    const handleHashChange = () => {
      const route = getRouteFromHash();
      setActiveRoute(route);
    };

    window.addEventListener('hashchange', handleHashChange);
    return () => window.removeEventListener('hashchange', handleHashChange);
  }, []);

  useEffect(() => {
    if (!currentDraft) {
      setDraft({
        schema_version: '1.0',
        draft_id: 'draft-101',
        tenant_id: 'default',
        name: 'Onboarding Flow',
        version: 1,
        nodes: [{ id: 'node-start', type: 'trigger', name: 'Start Event' }],
        edges: [],
      });
    }
  }, [currentDraft, setDraft]);
  const [selectedRunId, setSelectedRunId] = useState<string>('run-601');

  // Save Progress State
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [saveMessage, setSaveMessage] = useState<string>('');

  // Focus Return Ref for Shortcuts Modal
  const shortcutsBtnRef = useRef<HTMLButtonElement | null>(null);

  // Accessibility Announcer Hook
  const { announcement, announce } = useAnnouncer();
  // ETag Conflict Resolution State
  const [conflictServerDraft, setConflictServerDraft] = useState<GraphDraft | null>(null);
  const [isAgyModalOpen, setIsAgyModalOpen] = useState(false);


  const navItems: { route: NavigationRoute; label: string }[] = [
    { route: 'journeys', label: 'Journeys Directory' },
    { route: 'catalog', label: 'Component Catalog' },
    { route: 'history', label: 'Version History' },
    { route: 'runs', label: 'Execution Runs' },
    { route: 'static-lists', label: 'Static Lists' },
    { route: 'experiments', label: 'Experiment Analytics' },
    { route: 'canvas', label: 'Journey Canvas' },
  ];

  const routeMap: Record<NavigationRoute, string> = {
    journeys: 'Journeys Directory',
    catalog: 'Component Catalog',
    history: 'Version History',
    runs: 'Execution Runs',
    'run-detail': 'Run Details',
    canvas: 'Journey Canvas',
    experiments: 'Experiment Analytics',
    'static-lists': 'Static Lists Directory',
  };
  // Save Workflow with Conflict Simulation / Handling
  const handleSaveDraft = async () => {
    announce('Saving journey draft revision...', 'polite');

    try {
      const client = new JourneyApiClient();
      let savedDraft: GraphDraft;

      const tenantId = currentDraft?.tenant_id || 'default';

      if (currentDraft && currentDraft.draft_id) {
        try {
          const reqHeaders: Record<string, string> = { 'X-Tenant-ID': tenantId };
          if (currentDraft.content_hash && currentDraft.content_hash !== 'undefined' && currentDraft.content_hash !== 'null') {
            reqHeaders['If-Match'] = currentDraft.content_hash;
          }
          const res = await client.updateJourneyDraft(currentDraft.draft_id, currentDraft, reqHeaders);
          const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
          savedDraft = { ...res.draft, content_hash: updatedHash || res.draft.content_hash };
        } catch (apiErr: unknown) {
          if (apiErr instanceof APIError && apiErr.code === 'SIMULATED_API_FAILURE') {
            throw apiErr;
          }
          if (apiErr instanceof APIError && (apiErr.status === 409 || apiErr.status === 412)) {
            closeModal();
            setSaveStatus('idle');
            handleSimulateConflict();
            return;
          }
          if (apiErr instanceof APIError && apiErr.status === 404) {
            const res = await client.createJourneyDraft(currentDraft, {
              'X-Tenant-ID': tenantId,
            });
            const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
            savedDraft = { ...res.draft, content_hash: updatedHash || res.draft.content_hash };
          } else {
            throw apiErr;
          }
        }
      } else {
        const draftToSave: GraphDraft = {
          schema_version: '1.0',
          draft_id: `draft-${Date.now().toString().slice(-4)}`,
          tenant_id: tenantId,
          name: 'New Journey',
          version: 1,
          nodes: [{ id: 'node-start', type: 'trigger', name: 'Start Event' }],
          edges: [],
        };
        const res = await client.createJourneyDraft(draftToSave, {
          'X-Tenant-ID': tenantId,
        });
        const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
        savedDraft = { ...res.draft, content_hash: updatedHash || res.draft.content_hash };
      }
      setDraft(savedDraft);
      markSaved();
      queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
      setSaveStatus('idle');
      closeModal();
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.SUCCESS,
        title: 'Draft Saved Successfully',
        message: `Saved "${savedDraft.name}" (v${savedDraft.version}) successfully.`,
      });
      announce(`Journey draft saved successfully as version ${savedDraft.version}`, 'polite');
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to save journey draft';
      setSaveStatus('idle');
      closeModal();
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.ERROR,
        title: 'Failed to Save Draft',
        message: errMsg,
      });
      announce(`Failed to save draft: ${errMsg}`, 'assertive');
    }
  };

  const handlePublishDraft = async () => {
    announce('Publishing journey workflow revision to production engine...', 'polite');

    try {
      const client = new JourneyApiClient();
      const tenantId = currentDraft?.tenant_id || 'default';
      const draftId = currentDraft?.draft_id || 'draft-101';

      try {
        await client.updateJourneyDraft(
          draftId,
          currentDraft || {
            schema_version: '1.0',
            draft_id: draftId,
            tenant_id: tenantId,
            name: 'New Journey',
            version: 1,
            nodes: [],
            edges: [],
          },
          {
            'If-Match': currentDraft?.content_hash || '*',
            'X-Tenant-ID': tenantId,
          }
        );
      } catch (updateErr) {
        if (updateErr instanceof APIError && updateErr.code === 'SIMULATED_API_FAILURE') {
          throw updateErr;
        }
      }

      await client.publishJourneyDraft(draftId, { 'X-Tenant-ID': tenantId });

      setSaveStatus('idle');
      closeModal();
      queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
      setActiveRoute('journeys');
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.SUCCESS,
        title: 'Journey Workflow Published',
        message: `Successfully published "${currentDraft?.name || 'Journey Draft'}"! Active version ready.`,
      });
      announce(`Journey draft published successfully`, 'polite');
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to publish journey draft';
      setSaveStatus('idle');
      closeModal();
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.ERROR,
        title: 'Failed to Publish Workflow',
        message: errMsg,
      });
      announce(`Failed to publish draft: ${errMsg}`, 'assertive');
    }
  };

  const handleCreateJourneyTop = async () => {
    try {
      const client = new JourneyApiClient();
      const tenantId = 'default';
      const draftToSave: GraphDraft = {
        schema_version: '1.0',
        draft_id: `draft-${Date.now().toString().slice(-4)}`,
        tenant_id: tenantId,
        name: 'New Journey',
        version: 1,
        nodes: [{ id: 'node-start', type: 'trigger', name: 'Start Event' }],
        edges: [],
      };
      const res = await client.createJourneyDraft(draftToSave, {
        'X-Tenant-ID': tenantId,
      });
      setDraft(res.draft);
      queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
      setActiveRoute('canvas');
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.SUCCESS,
        title: 'New Journey Initialized',
        message: `Created "${res.draft.name}".`,
      });
      announce(`Created new journey ${res.draft.name}`, 'polite');
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Could not create new journey.';
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.ERROR,
        title: 'Failed to Create Journey',
        message: msg,
      });
      console.error('Failed to create new journey:', err);
      setActiveRoute('canvas');
    }
  };

  const handleSimulateConflict = () => {
    const draftBase = currentDraft || {
      schema_version: '1.0',
      draft_id: 'draft-101',
      tenant_id: 'default',
      name: 'User Onboarding Journey',
      version: 1,
      nodes: [{ id: 'node-1', type: 'EventStart', name: 'User Signup Event' }],
      edges: [],
    };
    const serverRevision: GraphDraft = {
      ...draftBase,
      version: (draftBase.version || 1) + 1,
      name: `${draftBase.name} (Remote Revision)`,
      nodes: [
        ...draftBase.nodes,
        {
          id: 'server-node-added',
          type: 'action',
          name: 'Remote Server Action',
          config: { recipient: 'remote@server.internal' },
        },
      ],
      content_hash: 'etag-hash-server-remote-998',
    };
    setConflictServerDraft(serverRevision);
    openModal('confirmDiscard'); // triggers conflict dialog
    announce('ETag conflict detected with remote server revision', 'assertive');
  };

  const [isDevPanelOpen, setIsDevPanelOpen] = useState(false);

  const [toastState, setToastState] = useState<{
    isOpen: boolean;
    messageType: ToastMessageType;
    title: string;
    message?: string;
  }>({
    isOpen: false,
    messageType: ToastMessageType.SUCCESS,
    title: '',
    message: '',
  });

  const handleStartTestRun = async (_config: {
    draftId: string;
    executionMode: 'realistic' | 'forced_variant_coverage';
    fakeProviders: string[];
    expiryHours: number;
    staticListId?: string;
  }) => {
    try {
      const client = new JourneyApiClient();
      const tenantId = currentDraft?.tenant_id || 'default';
      const draftId = currentDraft?.draft_id || _config.draftId || 'draft-101';

      // 0. Ensure active draft is persisted
      if (currentDraft && currentDraft.draft_id) {
        try {
          const reqHeaders: Record<string, string> = { 'X-Tenant-ID': tenantId, 'If-Match': '*' };
          const res = await client.updateJourneyDraft(currentDraft.draft_id, currentDraft, reqHeaders);
          const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
          setDraft({ ...res.draft, content_hash: updatedHash || res.draft.content_hash });
          markSaved();
        } catch (updateErr) {
          if (updateErr instanceof APIError && updateErr.code === 'SIMULATED_API_FAILURE') {
            throw updateErr;
          }
          try {
            const res = await client.createJourneyDraft(currentDraft, { 'X-Tenant-ID': tenantId });
            const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
            setDraft({ ...res.draft, content_hash: updatedHash || res.draft.content_hash });
            markSaved();
          } catch (createErr) {
            if (createErr instanceof APIError && createErr.code === 'SIMULATED_API_FAILURE') {
              throw createErr;
            }
            console.warn('Draft auto-save warning:', createErr);
          }
        }
      }

      let launchedRunId: string | undefined;

      // 1. Execute test run API request
      const res = await client.startTestRun(
        {
          draft_id: draftId,
          static_list_id: _config.staticListId,
          execution_mode: _config.executionMode,
          status: 'running',
          mock_inputs: {
            execution_mode: _config.executionMode,
            fake_providers: _config.fakeProviders,
            expiry_hours: _config.expiryHours,
          },
        },
        { 'X-Tenant-ID': tenantId }
      );

      if (res && typeof res === 'object') {
        launchedRunId = res.test_run_id || res.run_id;
      }

      setToastState({
        isOpen: true,
        messageType: ToastMessageType.SUCCESS,
        title: 'Journey Test Execution Successfully Launched',
        message: `Launched execution scenario for "${currentDraft?.name || draftId}".`,
      });
      announce('Test run started successfully', 'polite');

      if (launchedRunId) {
        setSelectedRunId(launchedRunId);
        setActiveRoute('run-detail');
      } else {
        setActiveRoute('runs');
      }
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : 'Could not launch execution run.';
      setToastState({
        isOpen: true,
        messageType: ToastMessageType.ERROR,
        title: 'Failed to Launch Test Execution',
        message: errMsg,
      });
      announce(`Failed to start test run: ${errMsg}`, 'assertive');
      console.error('Failed to start test run:', err);
    } finally {
      closeModal();
      queryClient.invalidateQueries({ queryKey: ['runs'] });
      queryClient.refetchQueries({ queryKey: ['runs'] });
    }
  };

  useEffect(() => {
    const handleGlobalKeyDown = (e: KeyboardEvent) => {
      // Do not trigger shortcuts when typing inside form inputs or textareas
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === 'INPUT' ||
          target.tagName === 'TEXTAREA' ||
          target.isContentEditable)
      ) {
        return;
      }

      // Developer Panel Shortcut: Cmd+? / Ctrl+?
      if ((e.metaKey || e.ctrlKey) && (e.key === '?' || (e.shiftKey && e.code === 'Slash'))) {
        e.preventDefault();
        setIsDevPanelOpen((prev) => !prev);
        announce('Toggled Developer Control Panel', 'polite');
        return;
      }

      // Help Shortcut: ? (without Cmd/Ctrl)
      if (e.key === '?' && !e.metaKey && !e.ctrlKey) {
        e.preventDefault();
        openModal('keyboardShortcuts');
        return;
      }

      // AGY Prompt Context Export Shortcut: Cmd+Shift+A
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'a') {
        e.preventDefault();
        setIsAgyModalOpen(true);
        announce('Opened Antigravity AI prompt context exporter', 'polite');
        return;
      }

      // Save Shortcut: Cmd+S / Ctrl+S
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
        e.preventDefault();
        handleSaveDraft();
        return;
      }

      // Undo Shortcut: Cmd+Z / Ctrl+Z (without shift)
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'z' && !e.shiftKey) {
        e.preventDefault();
        undo();
        announce('Undid last graph edit action', 'polite');
        return;
      }

      // Redo Shortcut: Cmd+Shift+Z / Ctrl+Y / Ctrl+Shift+Z
      if (
        ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'z') ||
        ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'y')
      ) {
        e.preventDefault();
        redo();
        announce('Redid graph edit action', 'polite');
        return;
      }

      // Escape: Close active modal or clear selection
      if (e.key === 'Escape') {
        if (activeModal) {
          closeModal();
        }
        return;
      }

      // Delete / Backspace: Delete selected elements on canvas
      if ((e.key === 'Delete' || e.key === 'Backspace') && activeRoute === 'canvas') {
        if (selectedNodeId || selectedEdgeId) {
          e.preventDefault();
          deleteElements(
            selectedNodeId ? [selectedNodeId] : [],
            selectedEdgeId ? [selectedEdgeId] : []
          );
          announce('Deleted selected graph element', 'polite');
        }
      }
    };

    window.addEventListener('keydown', handleGlobalKeyDown);
    return () => window.removeEventListener('keydown', handleGlobalKeyDown);
  }, [
    activeModal,
    activeRoute,
    selectedNodeId,
    selectedEdgeId,
    openModal,
    closeModal,
    undo,
    redo,
    deleteElements,
    announce,
  ]);

  return (
    <>
      <div className="flex h-screen bg-[#0B0F19] text-[#DFE2F1] font-['Outfit',sans-serif] overflow-hidden">
      <LiveAnnouncer announcement={announcement} />
      <Toast
        isOpen={toastState.isOpen}
        messageType={toastState.messageType}
        title={toastState.title}
        message={toastState.message}
        onClose={() => setToastState((prev) => ({ ...prev, isOpen: false }))}
        testId="app-toast-notification"
      />

      {/* Sidebar Navigation - Full Height */}
      {isSidebarOpen && (
        <aside
          aria-label="Main Sidebar Navigation"
          className="w-64 bg-[#0a0e18] border-r border-[#464554] flex flex-col h-full overflow-y-auto shrink-0 z-40"
        >
          {/* Top Brand & Action Header */}
          <div className="p-4 space-y-4">
            {/* Header Logo */}
            <div className="flex items-start gap-2.5 mb-2">
              <span className="material-symbols-outlined text-2xl text-[#c0c1ff] shrink-0 mt-0.5" style={{ fontVariationSettings: "'FILL' 1" }}>
                electric_bolt
              </span>
              <div className="flex-1 min-w-0">
                <h1 className="text-lg font-['Outfit'] font-bold text-[#dfe2f1] leading-tight">
                  Journey Control Engine
                </h1>
                <p className="text-[11px] font-mono text-[#c0c1ff] opacity-90 break-all leading-normal mt-0.5" data-testid="sidebar-version-tag">
                  {BUILD_TAG}
                </p>
              </div>
            </div>

            {/* Top Create Journey Button */}
            <Button
              onClick={handleCreateJourneyTop}
              data-testid="sidebar-create-journey-btn"
              variant="primary-purple"
              size="lg"
              icon="add"
              fullWidth
              className="mb-2"
            >
              New Journey
            </Button>
          </div>

          {/* Seamless Nav List (Flush with Sidebar) */}
          <nav className="w-full space-y-1 py-1">
            {navItems.map((item) => {
              const isActive = activeRoute === item.route;
              let icon = 'account_tree';
              if (item.route === 'journeys') icon = 'account_tree';
              if (item.route === 'catalog') icon = 'inventory_2';
              if (item.route === 'history') icon = 'history';
              if (item.route === 'runs') icon = 'list_alt';
              if (item.route === 'static-lists') icon = 'format_list_bulleted';
              if (item.route === 'experiments') icon = 'analytics';
              if (item.route === 'canvas') icon = 'science';

              return (
                <button
                  key={item.route}
                  onClick={() => setActiveRoute(item.route)}
                  aria-label={item.label}
                  className={`w-full flex items-center gap-3 px-6 py-3 font-medium text-sm transition-all duration-200 text-left cursor-pointer rounded-none border-none ${
                    isActive
                      ? 'sidebar-btn-active bg-[#b76dff]/20 text-[#ddb7ff] border-l-4 border-[#ddb7ff]'
                      : 'sidebar-btn-inactive text-[#c7c4d7] hover:text-[#dfe2f1] hover:bg-[#262a35]'
                  }`}
                  style={
                    isActive
                      ? {
                          backgroundColor: 'rgba(183, 109, 255, 0.2)',
                          color: '#ddb7ff',
                          borderLeft: '4px solid #ddb7ff',
                          borderRadius: '0px',
                        }
                      : {
                          backgroundColor: 'transparent',
                          color: '#c7c4d7',
                          borderLeft: '4px solid transparent',
                          borderRadius: '0px',
                        }
                  }
                >
                  <span
                    className={`material-symbols-outlined text-xl ${isActive ? 'text-[#ddb7ff]' : 'text-[#c7c4d7]'}`}
                    style={{ color: isActive ? '#ddb7ff' : '#c7c4d7' }}
                  >
                    {icon}
                  </span>
                  <span>{item.label}</span>
                </button>
              );
            })}
          </nav>

          {/* Sidebar Footer Info */}
          <div className="mt-auto p-4 border-t border-[#464554] space-y-1 text-xs font-['Outfit',sans-serif]">
            <div className="flex items-center justify-between text-[#908fa0] px-2 py-1">
              <span>Environment</span>
              <span className="px-2 py-0.5 rounded-none bg-amber-500/10 text-amber-400 border border-amber-500/20 text-[10px] font-mono font-semibold">TEST</span>
            </div>
          </div>
        </aside>
      )}

      {/* Main Workspace Column */}
      <div className="flex-1 flex flex-col min-w-0 h-full max-h-full min-h-0 overflow-hidden">
        {/* Top Header Bar */}
        <header
          role="banner"
          aria-label="Journey Engine Header"
          className="flex items-center justify-between px-6 py-3 bg-[#0F131D]/80 backdrop-blur-xl border-b border-[#464554] shadow-sm z-30 h-16 shrink-0 min-w-0 overflow-hidden w-full"
        >
          <div className="flex items-center gap-4 min-w-0 overflow-x-auto">
            {/* Breadcrumb Navigation */}
            <div className="flex items-center gap-2 text-sm font-['Outfit']">
              <span className="text-[#c7c4d7] font-medium hover:bg-white/5 px-2 py-1 rounded-none transition-colors cursor-pointer">Acme Corp</span>
              <span className="text-[#908fa0]">/</span>
              {activeRoute === 'canvas' ? (
                <InlineJourneyName
                  onRenameSuccess={(newName) => {
                    setToastState({
                      isOpen: true,
                      messageType: ToastMessageType.SUCCESS,
                      title: 'Journey Renamed Successfully',
                      message: `Journey workflow title updated to "${newName}".`,
                    });
                  }}
                  onRenameError={(errMsg) => {
                    setToastState({
                      isOpen: true,
                      messageType: ToastMessageType.ERROR,
                      title: 'Failed to Rename Journey',
                      message: errMsg,
                    });
                  }}
                />
              ) : (
                <span className="text-[#dfe2f1] font-semibold">{routeMap[activeRoute] || activeRoute}</span>
              )}
            </div>
          </div>

          {/* Top Header Controls */}
          <div className="flex items-center gap-3">
            {/* Search Input */}
            <div className="relative hidden md:block">
              <input
                type="text"
                placeholder="Search engine..."
                className="bg-[#171b26] border border-[#464554] rounded-none px-4 py-1.5 text-xs text-white w-64 focus:ring-2 focus:ring-[#c0c1ff] outline-none transition-all"
              />
            </div>


            <div className="w-8 h-8 rounded-none overflow-hidden border border-[#464554] shadow-md shrink-0" title="User Profile">
              <div className="w-full h-full bg-[#171b26] flex items-center justify-center text-[#c0c1ff] hover:text-white hover:bg-[#262a35] transition-colors cursor-pointer">
                <span className="material-symbols-outlined text-lg">person</span>
              </div>
            </div>
          </div>
        </header>

        {/* Content Area */}
        <main id="main-content" role="main" aria-label="Main Content Area" className="flex-1 flex flex-row min-w-0 h-full max-h-full min-h-0 bg-[#0B0F19] overflow-hidden relative">
          {/* Route Rendering */}
          {activeRoute === 'journeys' && (
            <JourneysPage
              onSelectJourney={async (draftId) => {
                try {
                  const client = new JourneyApiClient();
                  const { draft: fetched, etag } = await client.getJourneyDraft(draftId, {
                    'X-Tenant-ID': 'default',
                  });
                  const updatedHash = etag ? etag.replace(/^W\//i, '').replace(/"/g, '') : fetched.content_hash;
                  setDraft({ ...fetched, content_hash: updatedHash || fetched.content_hash });
                  setActiveRoute('canvas');
                } catch (err: unknown) {
                  const msg = err instanceof Error ? err.message : 'API connection error';
                  setToastState({
                    isOpen: true,
                    messageType: ToastMessageType.ERROR,
                    title: 'Failed to Open Journey',
                    message: `Could not fetch draft "${draftId}": ${msg}`,
                  });
                  announce(`Failed to fetch journey draft "${draftId}": ${msg}`, 'assertive');
                }
              }}
            />
          )}

          {activeRoute === 'catalog' && <CatalogPage />}

          {activeRoute === 'history' && <VersionHistoryPage />}

          {activeRoute === 'runs' && (
            <RunListPage
              onSelectRun={(runId) => {
                setSelectedRunId(runId);
                setActiveRoute('run-detail');
              }}
            />
          )}

          {activeRoute === 'run-detail' && (
            <RunDetailPage
              runId={selectedRunId}
              onBackToList={() => setActiveRoute('runs')}
            />
          )}
          {activeRoute === 'experiments' && <ExperimentReportView />}

          {activeRoute === 'static-lists' && (
            <StaticListsPage onOpenUpload={() => openModal('staticListUpload')} />
          )}

          {activeRoute === 'canvas' && (
            <div className="flex-1 flex flex-row min-w-0 w-full h-full max-h-full min-h-0 relative overflow-hidden bg-[#0B0F19]">
              {/* Desktop-Required Notice for Canvas Editor at screen width < 1024px */}
              <DesktopOnlyNotice onNavigateToList={() => setActiveRoute('journeys')} />

              <div className="flex-1 flex flex-col min-w-0 w-full h-full max-h-full min-h-0 relative overflow-hidden">
                <Canvas
                  onSaveDraft={handleSaveDraft}
                  onSimulateConflict={handleSimulateConflict}
                  onPublish={handlePublishDraft}
                  onUploadStaticList={() => openModal('staticListUpload')}
                  onLaunchTestRun={() => openModal('testRunConfig')}
                  onKeyboardShortcuts={() => openModal('keyboardShortcuts')}
                  onAgyContext={() => setIsAgyModalOpen(true)}
                />
              </div>
              {/* Inspector Drawer for Canvas */}
              {isInspectorOpen && (
                <div className="h-full max-h-full min-h-0 shrink-0 z-20">
                  <NodeInspector nodeId={selectedNodeId} onClose={toggleInspector} />
                </div>
              )}
            </div>
          )}
        </main>
      </div>

      {/* Keyboard Shortcuts Modal */}
      <KeyboardShortcutsModal
        isOpen={activeModal === 'keyboardShortcuts'}
        onClose={closeModal}
        invokingControlRef={shortcutsBtnRef}
      />

      {/* Antigravity AI Prompt Context Exporter Modal */}
      <AgyContextModal
        isOpen={isAgyModalOpen}
        onClose={() => setIsAgyModalOpen(false)}
      />

      {/* ETag Conflict Modal */}
      {activeModal === 'confirmDiscard' && conflictServerDraft && (
        <ConflictDialog
          isOpen={true}
          localDraft={currentDraft}
          serverDraft={conflictServerDraft}
          localETag={currentDraft?.content_hash || 'etag-local-1'}
          serverETag={conflictServerDraft.content_hash || 'etag-server-2'}
          onKeepLocal={async () => {
            closeModal();
            setConflictServerDraft(null);
            if (currentDraft && currentDraft.draft_id) {
              try {
                const client = new JourneyApiClient();
                const tenantId = currentDraft.tenant_id || 'default';
                const res = await client.updateJourneyDraft(currentDraft.draft_id, currentDraft, {
                  'X-Tenant-ID': tenantId,
                });
                const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
                const saved = { ...res.draft, content_hash: updatedHash || res.draft.content_hash };
                setDraft(saved);
                markSaved();
                queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
                announce('Overwrote server with local edits successfully', 'polite');
              } catch (err) {
                console.error('Failed to overwrite server draft:', err);
              }
            }
          }}
          onAcceptServer={() => {
            setDraft(conflictServerDraft);
            markSaved();
            queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
            closeModal();
            setConflictServerDraft(null);
          }}
          onMerge={async () => {
            if (currentDraft && conflictServerDraft) {
              const mergedNodes = [...currentDraft.nodes, ...conflictServerDraft.nodes.filter((sn: GraphNode) => !currentDraft.nodes.some((ln: GraphNode) => ln.id === sn.id))];
              const mergedDraft = {
                ...currentDraft,
                nodes: mergedNodes,
              };
              setDraft(mergedDraft);
              try {
                const client = new JourneyApiClient();
                const tenantId = currentDraft.tenant_id || 'default';
                const res = await client.updateJourneyDraft(currentDraft.draft_id, mergedDraft, {
                  'X-Tenant-ID': tenantId,
                });
                const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
                const saved = { ...res.draft, content_hash: updatedHash || res.draft.content_hash };
                setDraft(saved);
                markSaved();
                queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
                announce('Merged local and server revisions successfully', 'polite');
              } catch (err) {
                console.error('Failed to save merged draft:', err);
              }
            }
            closeModal();
            setConflictServerDraft(null);
          }}
          onCancel={() => {
            closeModal();
            setConflictServerDraft(null);
          }}
        />
      )}
      {/* Static List CSV Upload Modal */}
      <StaticListUploadModal
        isOpen={activeModal === 'staticListUpload'}
        onClose={closeModal}
        onUploadSuccess={(uploadedData) => {
          queryClient.invalidateQueries({ queryKey: ['static-lists'] });
          queryClient.invalidateQueries({ queryKey: ['static-lists-directory'] });
          setToastState({
            isOpen: true,
            messageType: ToastMessageType.SUCCESS,
            title: 'Static Audience List Uploaded',
            message: uploadedData?.name
              ? `Uploaded list "${uploadedData.name}" (${uploadedData.rowCount || uploadedData.item_count || 0} members).`
              : 'CSV audience members ingested successfully.',
          });
          announce('Static list uploaded successfully', 'polite');
          closeModal();
        }}
        onUploadError={(errMsg) => {
          setToastState({
            isOpen: true,
            messageType: ToastMessageType.ERROR,
            title: 'Static List Upload Failed',
            message: errMsg,
          });
          announce(`Static list upload failed: ${errMsg}`, 'assertive');
        }}
      />

      {/* Configure Test Run Execution Modal */}
      <TestRunModal
        isOpen={activeModal === 'testRunConfig' || activeModal === 'simulationConfig'}
        onClose={closeModal}
        draftId={currentDraft?.draft_id || 'draft-101'}
        onStartTestRun={handleStartTestRun}
      />

      {/* Developer Control Panel */}
      <DeveloperPanel
        isOpen={isDevPanelOpen}
        onClose={() => setIsDevPanelOpen(false)}
        onFireToast={(type, title, message) => {
          setToastState({
            isOpen: true,
            messageType: type,
            title,
            message,
          });
        }}
      />

      {activeModal &&
        activeModal !== 'confirmDiscard' &&
        activeModal !== 'keyboardShortcuts' &&
        activeModal !== 'staticListUpload' &&
        activeModal !== 'testRunConfig' &&
        activeModal !== 'simulationConfig' && (
          <Modal
            isOpen={true}
            onClose={() => {
              closeModal();
              setSaveStatus('idle');
            }}
            title={
              activeModal === 'save'
                ? saveStatus === 'saving'
                  ? 'Saving Journey Draft...'
                  : saveStatus === 'success'
                  ? 'Draft Saved!'
                  : 'Save Error'
                : activeModal === 'publish'
                ? saveStatus === 'saving'
                  ? 'Publishing Journey Workflow...'
                  : saveStatus === 'success'
                  ? 'Workflow Published!'
                  : 'Publish Error'
                : `Modal: ${activeModal.toUpperCase()}`
            }
            subtitle={
              activeModal === 'save' || activeModal === 'publish'
                ? saveMessage || 'Processing workflow revision on server...'
                : 'Perform action for modal target.'
            }
            maxWidth="sm"
            footer={
              <Button
                onClick={() => {
                  closeModal();
                  setSaveStatus('idle');
                }}
                variant="secondary"
              >
                Close
              </Button>
            }
          >
            <p className="text-sm font-['Outfit'] text-[#908fa0]">
              {activeModal === 'save' || activeModal === 'publish'
                ? saveMessage || 'Processing workflow revision on server...'
                : 'Perform action for modal target.'}
            </p>
          </Modal>
        )}
      </div>
    </>
  );
}

export function App() {
  const [paramFieldKey, setParamFieldKey] = useState<string | null>(null);

  useEffect(() => {
    const handler = (e: Event) => {
      const customEvt = e as CustomEvent<{ fieldKey: string }>;
      if (customEvt.detail?.fieldKey) {
        setParamFieldKey(customEvt.detail.fieldKey);
      }
    };
    window.addEventListener('open-param-modal', handler);
    return () => window.removeEventListener('open-param-modal', handler);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>
        <DashboardContent />
        <ParameterModal
          isOpen={Boolean(paramFieldKey)}
          onClose={() => setParamFieldKey(null)}
          targetFieldLabel={paramFieldKey || undefined}
          onSelectToken={(token) => {
            if (paramFieldKey) {
              window.dispatchEvent(new CustomEvent('select-param-token', { detail: { token, fieldKey: paramFieldKey } }));
            }
            setParamFieldKey(null);
          }}
        />
      </ErrorBoundary>
    </QueryClientProvider>
  );
}

export default App;
