import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { JourneyApiClient } from '../api/client';
import { Skeleton } from '../components/Skeleton';
import { DegradedStateView } from '../components/DegradedStateView';
import { useRouteParams } from '../hooks/useRouteParams';
import type { DraftVersionItem } from '../types/api';

const apiClient = new JourneyApiClient();

const DEFAULT_VERSION_HISTORY_PARAMS = {
  draftId: 'draft-101',
  v1: '1',
  v2: '2',
};

export const VersionHistoryPage: React.FC = () => {
  const [params, setParams] = useRouteParams(DEFAULT_VERSION_HISTORY_PARAMS);

  const draftId = params.draftId || 'draft-101';

  // Fetch Version History List for given Draft ID
  const {
    data: versionData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['versions', draftId],
    queryFn: async () => {
      const res = await apiClient.listDraftVersions(draftId);
      return res;
    },
  });

  const versions: DraftVersionItem[] = versionData?.versions || [];

  const selectedV1 = versions.find((v) => String(v.version) === params.v1) || versions[0];
  const selectedV2 = versions.find((v) => String(v.version) === params.v2) || versions[1] || versions[0];

  return (
    <div className="w-full h-full flex flex-col p-6 bg-[#0B0F19] overflow-y-auto space-y-6 text-[#dfe2f1]">
      {/* Top Section Header */}
      <div className="flex flex-wrap items-center justify-between gap-4 shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight flex items-center gap-3">
            <span className="material-symbols-outlined text-[#ddb7ff] text-2xl">history</span>
            Version History & Canonical IR Diffs
          </h1>
          <p className="text-xs text-[#908fa0] mt-1">
            Audit immutable revision history, inspect hash checksums, and evaluate IR graph diffs.
          </p>
        </div>
      </div>

      {/* Filter & Draft Selection Bar */}
      <div className="w-full p-4 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl flex flex-wrap items-center justify-between gap-4 shrink-0">
        <div className="flex items-center gap-3">
          <label className="text-xs font-semibold text-[#908fa0] uppercase tracking-wider font-mono">
            Draft ID:
          </label>
          <input
            type="text"
            value={draftId}
            onChange={(e) => setParams({ draftId: e.target.value })}
            className="px-4 py-2 rounded-none bg-[#171b26] border border-[#464554] text-white placeholder-[#908fa0] text-xs focus:outline-none focus:border-[#c0c1ff] focus:ring-1 focus:ring-[#c0c1ff] font-mono w-48"
          />
        </div>

        <div className="flex items-center gap-2 text-xs text-[#908fa0]">
          <span>Comparing:</span>
          <span className="px-2.5 py-1 rounded-none bg-[#171b26] border border-[#464554] text-[#ddb7ff] font-mono font-semibold">
            v{selectedV1?.version || 1}
          </span>
          <span>➔</span>
          <span className="px-2.5 py-1 rounded-none bg-[#171b26] border border-[#464554] text-[#6ee7b7] font-mono font-semibold">
            v{selectedV2?.version || 2}
          </span>
        </div>
      </div>

      {/* Error State with Retry Trigger */}
      {isError && (
        <DegradedStateView
          type="api-disconnected"
          title={`Failed to Load Version History for "${draftId}"`}
          description={(error as Error)?.message || 'Could not fetch revision history from backend service.'}
          actionLabel="Retry Fetching"
          onRetry={() => { refetch(); }}
        />
      )}

      {/* Main Content Layout */}
      {isLoading ? (
        <div className="p-6 rounded-none bg-[#0F131D]/90 border border-[#464554] shadow-xl">
          <Skeleton count={4} height="3rem" />
        </div>
      ) : versions.length === 0 ? (
        <DegradedStateView
          type="stale-report"
          title="No Version History Found"
          description={`Draft "${draftId}" has no recorded versions in the canonical repository.`}
        />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 w-full items-start">
          {/* Left Column: Revision Timeline */}
          <div className="lg:col-span-1 p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-[#464554] pb-3">
              <h2 className="text-base font-bold text-white flex items-center gap-2">
                <span className="material-symbols-outlined text-[#c0c1ff] text-lg">view_timeline</span>
                Immutable Revisions
              </h2>
              <span className="px-2 py-0.5 rounded-none bg-[#171b26] border border-[#464554] text-[#908fa0] text-xs font-mono">
                {versions.length} total
              </span>
            </div>

            <div className="space-y-3">
              {versions.map((ver) => {
                const isSelectedV1 = String(ver.version) === params.v1;
                const isSelectedV2 = String(ver.version) === params.v2;
                const isActive = isSelectedV1 || isSelectedV2;

                return (
                  <div
                    key={ver.version}
                    className={`p-4 rounded-none border transition-all space-y-3 ${
                      isActive
                        ? 'border-[#ddb7ff] bg-[#1c1f2a] ring-1 ring-[#ddb7ff]/30 shadow-lg'
                        : 'border-[#464554] bg-[#171b26] hover:border-[#c0c1ff] hover:bg-[#1f2434]'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-bold text-white font-mono">
                          Revision v{ver.version}
                        </span>
                        {isSelectedV1 && (
                          <span className="px-2 py-0.5 rounded-none bg-[#b76dff]/20 text-[#ddb7ff] border border-[#ddb7ff]/30 text-[10px] font-mono font-bold">
                            Base
                          </span>
                        )}
                        {isSelectedV2 && (
                          <span className="px-2 py-0.5 rounded-none bg-[#10b981]/20 text-[#6ee7b7] border border-[#10b981]/30 text-[10px] font-mono font-bold">
                            Target
                          </span>
                        )}
                      </div>
                      <span className="text-[11px] text-[#908fa0] font-mono">
                        {new Date(ver.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </span>
                    </div>

                    <div className="flex items-center gap-2 text-xs text-[#908fa0]">
                      <span className="font-semibold text-[#dfe2f1]">IR Hash:</span>
                      <code className="px-2 py-0.5 rounded-none bg-[#0B0F19] border border-[#464554] text-[#c0c1ff] font-mono text-[11px] truncate">
                        {ver.content_hash}
                      </code>
                    </div>

                    {/* Comparison Select Buttons */}
                    <div className="flex items-center gap-2.5 pt-3 border-t border-white/5">
                      <button
                        type="button"
                        onClick={() => setParams({ v1: String(ver.version) })}
                        className={`flex-1 px-3 py-2 rounded-none text-xs font-semibold font-['Outfit'] transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
                          isSelectedV1
                            ? 'bg-[#b76dff]/20 text-[#ddb7ff] border border-[#ddb7ff]/60 shadow-md font-bold hover:brightness-125'
                            : 'bg-[#171b26] text-[#908fa0] border border-[#464554] hover:text-[#dfe2f1] hover:border-[#c0c1ff] hover:bg-[#262a35] hover:brightness-125'
                        }`}
                      >
                        <span className="material-symbols-outlined text-xs">tune</span>
                        <span>{isSelectedV1 ? `Base (v${ver.version})` : 'Set as Base'}</span>
                      </button>
                      <button
                        type="button"
                        onClick={() => setParams({ v2: String(ver.version) })}
                        className={`flex-1 px-3 py-2 rounded-none text-xs font-semibold font-['Outfit'] transition-all cursor-pointer flex items-center justify-center gap-1.5 ${
                          isSelectedV2
                            ? 'bg-[#10b981]/20 text-[#6ee7b7] border border-[#10b981]/60 shadow-md font-bold hover:brightness-125'
                            : 'bg-[#171b26] text-[#908fa0] border border-[#464554] hover:text-[#dfe2f1] hover:border-[#6ee7b7] hover:bg-[#262a35] hover:brightness-125'
                        }`}
                      >
                        <span className="material-symbols-outlined text-xs">compare_arrows</span>
                        <span>{isSelectedV2 ? `Target (v${ver.version})` : 'Set as Target'}</span>
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Right Column: Revision Diff & Hash Verification */}
          <div className="lg:col-span-2 p-5 rounded-none bg-[#0F131D]/90 backdrop-blur-xl border border-[#464554] shadow-xl space-y-5">
            <div className="flex items-center justify-between border-b border-[#464554] pb-3">
              <h2 className="text-base font-bold text-white flex items-center gap-2">
                <span className="material-symbols-outlined text-[#ddb7ff] text-lg">difference</span>
                Revision Diff: v{selectedV1?.version || 1} ➔ v{selectedV2?.version || 2}
              </h2>
            </div>

            {/* Hash Checksum Card */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-[#171b26] p-4 rounded-none border border-[#464554]">
              <div className="space-y-1">
                <span className="text-xs font-semibold text-[#ddb7ff] flex items-center gap-1.5 font-mono">
                  <span className="w-2 h-2 rounded-none bg-[#ddb7ff]"></span>
                  Base Revision (v{selectedV1?.version})
                </span>
                <div className="text-[11px] text-[#908fa0]">Content Hash Checksum:</div>
                <code className="block p-2 rounded-none bg-[#0B0F19] border border-[#464554] text-[#c0c1ff] font-mono text-xs break-all">
                  {selectedV1?.content_hash || 'hash-v1'}
                </code>
              </div>

              <div className="space-y-1">
                <span className="text-xs font-semibold text-[#6ee7b7] flex items-center gap-1.5 font-mono">
                  <span className="w-2 h-2 rounded-none bg-[#6ee7b7]"></span>
                  Target Revision (v{selectedV2?.version})
                </span>
                <div className="text-[11px] text-[#908fa0]">Content Hash Checksum:</div>
                <code className="block p-2 rounded-none bg-[#0B0F19] border border-[#464554] text-[#6ee7b7] font-mono text-xs break-all">
                  {selectedV2?.content_hash || 'hash-v2'}
                </code>
              </div>
            </div>

            {/* Revision Summary & Hash Verification */}
            <div className="space-y-3">
              <h3 className="text-xs font-semibold text-[#908fa0] uppercase tracking-wider font-mono">
                Revision Details & Verification
              </h3>

              <div className="border border-[#464554] rounded-none overflow-hidden bg-[#171b26] p-4 space-y-3 text-xs">
                <div className="flex items-center justify-between">
                  <span className="text-[#908fa0] font-mono">Selected Base Revision:</span>
                  <span className="font-mono text-[#ddb7ff] font-bold">v{selectedV1?.version ?? 'N/A'}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[#908fa0] font-mono">Base Updated At:</span>
                  <span className="font-mono text-[#dfe2f1]">
                    {selectedV1?.updated_at ? new Date(selectedV1.updated_at).toLocaleString() : 'N/A'}
                  </span>
                </div>
                <div className="flex items-center justify-between border-t border-white/5 pt-3">
                  <span className="text-[#908fa0] font-mono">Selected Target Revision:</span>
                  <span className="font-mono text-[#6ee7b7] font-bold">v{selectedV2?.version ?? 'N/A'}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[#908fa0] font-mono">Target Updated At:</span>
                  <span className="font-mono text-[#dfe2f1]">
                    {selectedV2?.updated_at ? new Date(selectedV2.updated_at).toLocaleString() : 'N/A'}
                  </span>
                </div>
                <div className="border-t border-white/5 pt-3 flex items-center justify-between">
                  <span className="text-[#908fa0] font-mono">Checksum Status:</span>
                  <span className={`font-mono font-bold ${
                    selectedV1 && selectedV2 && selectedV1.content_hash === selectedV2.content_hash
                      ? 'text-emerald-400'
                      : 'text-amber-400'
                  }`}>
                    {selectedV1 && selectedV2 && selectedV1.content_hash === selectedV2.content_hash
                      ? 'Identical Hashes (No Changes)'
                      : 'Revision Hash Delta Detected'}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default VersionHistoryPage;
