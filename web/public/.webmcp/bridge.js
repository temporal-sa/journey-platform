// Cloudflare WebMCP Bridge Script for Local & Production Deployment
// Registers Model Context Protocol (MCP) tools on document.modelContext

(async function initWebMCP() {
  const currentScript = document.currentScript || document.querySelector('script[src*="bridge.js"]');
  const mcpUrl = currentScript?.getAttribute('data-mcp-url') || '/mcp';
  const packs = (currentScript?.getAttribute('data-packs') || 'mcp-server-client').split(',');

  console.log(`[WebMCP Bridge] Initializing WebMCP bridge (MCP Endpoint: ${mcpUrl}, Packs: ${packs.join(', ')})`);

  // Initialize polyfill/fallback for document.modelContext if browser hasn't enabled Chrome 146+ flag natively
  if (!document.modelContext) {
    const registeredTools = new Map();
    document.modelContext = {
      registerTool(tool) {
        if (!tool || !tool.name || typeof tool.execute !== 'function') {
          console.warn('[WebMCP] Invalid tool registration attempt:', tool);
          return;
        }
        registeredTools.set(tool.name, tool);
        console.log(`[WebMCP] Registered tool: ${tool.name}`);
      },
      unregisterTool(name) {
        registeredTools.delete(name);
      },
      getTools() {
        return Array.from(registeredTools.values()).map(t => ({
          name: t.name,
          description: t.description,
          inputSchema: t.inputSchema,
        }));
      },
      async callTool(name, args) {
        const tool = registeredTools.get(name);
        if (!tool) {
          throw new Error(`[WebMCP] Tool '${name}' not found`);
        }
        return await tool.execute(args);
      }
    };
  }

  // Register Site MCP Server pack tools via /mcp JSON-RPC endpoint
  if (packs.includes('mcp-server-client')) {
    try {
      const response = await fetch(mcpUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: 1,
          method: 'tools/list',
          params: {}
        })
      });

      if (!response.ok) {
        console.warn(`[WebMCP] Failed to fetch tools from ${mcpUrl}: HTTP ${response.status}`);
        return;
      }

      const payload = await response.json();
      const tools = payload?.result?.tools || [];

      console.log(`[WebMCP] Discovered ${tools.length} site MCP tools from ${mcpUrl}`);

      for (const tool of tools) {
        document.modelContext.registerTool({
          name: tool.name,
          description: tool.description,
          inputSchema: tool.inputSchema,
          execute: async (args) => {
            console.log(`[WebMCP] Executing tool '${tool.name}' with args:`, args);
            const callRes = await fetch(mcpUrl, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              credentials: 'same-origin',
              body: JSON.stringify({
                jsonrpc: '2.0',
                id: Date.now(),
                method: 'tools/call',
                params: {
                  name: tool.name,
                  arguments: args || {}
                }
              })
            });
            const callPayload = await callRes.json();
            if (callPayload.error) {
              throw new Error(callPayload.error.message || `Error executing ${tool.name}`);
            }
            return callPayload.result;
          }
        });
      }
    } catch (err) {
      console.error('[WebMCP] Error initializing Site MCP tools:', err);
    }
  }
})();
