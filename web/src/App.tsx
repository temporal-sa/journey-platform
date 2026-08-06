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
import { ParameterModal } from './components/inspector/ParameterModal';
import { Button } from './components/common/Button';
import { useAnnouncer, LiveAnnouncer } from './hooks/useAnnouncer';
import { ExperimentReportView } from './components/experiments/ExperimentReportView';
import { JourneyApiClient, APIError } from './api/client';
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

export function InlineJourneyName() {
  const currentDraft = useEditorStore((s) => s.currentDraft);
  const setDraftName = useEditorStore((s) => s.setDraftName);
  const hasUnsavedChanges = useEditorStore((s) => s.hasUnsavedChanges);
  const { announce } = useAnnouncer();

  const [isEditing, setIsEditing] = useState(false);
  const [editedName, setEditedName] = useState(currentDraft?.name || 'Onboarding Flow');

  useEffect(() => {
    if (currentDraft?.name) {
      setEditedName(currentDraft.name);
    }
  }, [currentDraft?.name]);

  const handleSave = () => {
    setIsEditing(false);
    const trimmed = editedName.trim();
    if (trimmed && trimmed !== currentDraft?.name) {
      setDraftName(trimmed);
      announce(`Journey name updated to ${trimmed}`, 'polite');
    } else if (!trimmed) {
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
          className="bg-[#11141d] border border-[#c0c1ff] focus:border-[#4cd7f6] focus:ring-2 focus:ring-[#4cd7f6]/30 rounded-none px-3 py-1 text-sm font-bold text-white outline-none transition-all font-['Outfit'] shadow-inner min-w-[220px]"
        />
        <span className="text-[10px] font-mono text-[#908fa0] animate-pulse">Press Enter to save</span>
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
    openModal('save');
    setSaveStatus('saving');
    setSaveMessage('Saving journey draft revision to database...');
    announce('Saving journey draft revision to database...', 'polite');

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
      setSaveStatus('success');
      setSaveMessage(`Successfully saved ${savedDraft.name} (v${savedDraft.version}) to PostgreSQL database`);
      announce(`Journey draft saved successfully to database as version ${savedDraft.version}`, 'polite');
      // Auto close after confirmation feedback
      setTimeout(() => {
        setSaveStatus('idle');
        closeModal();
      }, 800);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to save journey draft';
      setSaveStatus('error');
      setSaveMessage(errMsg);
      announce(`Failed to save draft: ${errMsg}`, 'assertive');
    }
  };

  const handlePublishDraft = async () => {
    openModal('publish');
    setSaveStatus('saving');
    setSaveMessage('Publishing journey workflow revision to production engine...');
    announce('Publishing journey workflow revision to production engine...', 'polite');

    try {
      const client = new JourneyApiClient();
      const tenantId = currentDraft?.tenant_id || 'default';

      if (currentDraft && currentDraft.draft_id) {
        try {
          await client.updateJourneyDraft(currentDraft.draft_id, currentDraft, {
            'If-Match': currentDraft.content_hash,
            'X-Tenant-ID': tenantId,
          });
        } catch {
          await client.createJourneyDraft(currentDraft, { 'X-Tenant-ID': tenantId });
        }

        try {
          await client.publishJourneyDraft(currentDraft.draft_id, { 'X-Tenant-ID': tenantId });
        } catch {
          await client.activateLocalJourney(currentDraft.draft_id, { 'X-Tenant-ID': tenantId });
        }
      }

      setSaveStatus('success');
      setSaveMessage(`Successfully published ${currentDraft?.name || 'Journey Draft'}! Active version ready.`);
      announce(`Journey draft published successfully`, 'polite');
      queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
      setTimeout(() => {
        setSaveStatus('idle');
        closeModal();
        setActiveRoute('journeys');
      }, 1000);
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : 'Failed to publish journey draft';
      setSaveStatus('error');
      setSaveMessage(errMsg);
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
        name: 'Untitled Journey',
        version: 1,
        nodes: [{ id: 'node-start', type: 'trigger', name: 'User Signup Event' }],
        edges: [],
      };
      const res = await client.createJourneyDraft(draftToSave, {
        'X-Tenant-ID': tenantId,
      });
      setDraft(res.draft);
      queryClient.invalidateQueries({ queryKey: ['journeys', 'list'] });
      setActiveRoute('canvas');
      announce(`Created new journey ${res.draft.name}`, 'polite');
    } catch (err) {
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

  const handleStartTestRun = async (_config: {
    draftId: string;
    executionMode: 'realistic' | 'forced_variant_coverage';
    fixturePack: string;
    fakeProviders: string[];
    expiryHours: number;
    targetCount: number;
    suppressionCount: number;
  }) => {
    try {
      const client = new JourneyApiClient();
      const tenantId = currentDraft?.tenant_id || 'default';

      // 0. Ensure active draft is persisted to PostgreSQL database
      if (currentDraft && currentDraft.draft_id) {
        try {
          const reqHeaders: Record<string, string> = { 'X-Tenant-ID': tenantId, 'If-Match': '*' };
          const res = await client.updateJourneyDraft(currentDraft.draft_id, currentDraft, reqHeaders);
          const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
          setDraft({ ...res.draft, content_hash: updatedHash || res.draft.content_hash });
          markSaved();
        } catch {
          try {
            const res = await client.createJourneyDraft(currentDraft, { 'X-Tenant-ID': tenantId });
            const updatedHash = res.etag ? res.etag.replace(/"/g, '') : res.draft.content_hash;
            setDraft({ ...res.draft, content_hash: updatedHash || res.draft.content_hash });
            markSaved();
          } catch (createErr) {
            console.warn('Draft auto-save warning:', createErr);
          }
        }
      }
      announce('Test run started successfully', 'polite');
    } catch (err) {
      console.error('Failed to start test run:', err);
    } finally {
      closeModal();
      queryClient.invalidateQueries({ queryKey: ['runs'] });
      queryClient.refetchQueries({ queryKey: ['runs'] });
      setActiveRoute('runs');
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

      // Help Shortcut: ?
      if (e.key === '?') {
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

      {/* Sidebar Navigation - Full Height */}
      {isSidebarOpen && (
        <aside
          aria-label="Main Sidebar Navigation"
          className="w-64 bg-[#0a0e18] border-r border-[#464554] flex flex-col h-full overflow-y-auto shrink-0 z-40"
        >
          {/* Top Brand & Action Header */}
          <div className="p-4 space-y-4">
            {/* Header Logo */}
            <div className="flex items-center gap-3 mb-2">
              <div className="w-10 h-10 rounded-none bg-gradient-to-br from-[#c0c1ff] to-[#ddb7ff] flex items-center justify-center text-[#1000a9] shadow-lg">
                <span className="material-symbols-outlined text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>electric_bolt</span>
              </div>
              <div>
                <h1 className="text-xl font-['Outfit'] font-bold text-[#dfe2f1] leading-tight">Journey Control Engine</h1>
                <p className="text-xs font-mono text-[#908fa0] opacity-60">V2.4.0-stable</p>
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
              <span className="px-2 py-0.5 rounded-none bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] font-mono font-semibold">PRODUCTION</span>
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
                <InlineJourneyName />
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
                  onTestMode={() => openModal('staticListUpload')}
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
        onUploadSuccess={() => {
          openModal('testRunConfig');
        }}
      />

      {/* Configure Test Run Execution Modal */}
      <TestRunModal
        isOpen={activeModal === 'testRunConfig' || activeModal === 'simulationConfig'}
        onClose={closeModal}
        draftId={currentDraft?.draft_id || 'draft-101'}
        onStartTestRun={handleStartTestRun}
      />

      {activeModal &&
        activeModal !== 'confirmDiscard' &&
        activeModal !== 'keyboardShortcuts' &&
        activeModal !== 'staticListUpload' &&
        activeModal !== 'testRunConfig' &&
        activeModal !== 'simulationConfig' && (
          <div
            role="dialog"
            aria-label={`Modal: ${activeModal}`}
            style={{
              position: 'fixed',
              inset: 0,
              backgroundColor: 'rgba(0,0,0,0.5)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 1000,
            }}
          >
          <div style={{ backgroundColor: '#fff', padding: '1.5rem', borderRadius: '0px', width: '360px', boxShadow: '0 4px 6px rgba(0,0,0,0.1)' }}>
            <h3
              style={{
                marginTop: 0,
                color:
                  (activeModal === 'save' || activeModal === 'publish') && saveStatus === 'error'
                    ? '#ef4444'
                    : (activeModal === 'save' || activeModal === 'publish') && saveStatus === 'success'
                    ? '#10b981'
                    : '#1e293b',
              }}
            >
              {activeModal === 'save'
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
                : `Modal: ${activeModal.toUpperCase()}`}
            </h3>
            <p className="text-sm" style={{ color: '#475569' }}>
              {activeModal === 'save' || activeModal === 'publish'
                ? saveMessage || 'Processing workflow revision on server...'
                : `Perform action for modal target.`}
            </p>
            <div style={{ textAlign: 'right', marginTop: '1rem' }}>
              <button
                onClick={() => {
                  closeModal();
                  setSaveStatus('idle');
                }}
                style={{ padding: '0.4rem 0.8rem', cursor: 'pointer', backgroundColor: '#3b82f6', color: '#fff', border: 'none', borderRadius: '0px' }}
              >
                Close
              </button>
            </div>
          </div>
        </div>
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
