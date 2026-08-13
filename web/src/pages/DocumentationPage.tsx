import React, { useState, useMemo } from 'react';
import { marked } from 'marked';
import { DirectoryLayout } from '../components/common/DirectoryLayout';
import { useRouteParams } from '../hooks/useRouteParams';

// Configure marked options
marked.setOptions({
  gfm: true,
  breaks: true,
});

const overviewMarkdown = `# User Guide: Event-Driven Customer Journey Engine

Welcome to the **User Guide** for the Event-Driven Customer Journey Engine. This documentation provides a comprehensive guide for marketers, product managers, developers, and operations teams to design, configure, test, monitor, and manage automated multi-channel customer journeys.

---

## System Overview & Key Concepts

The Event-Driven Customer Journey Engine is an enterprise customer engagement platform powered by high-throughput event processing and deterministic workflow execution. It enables teams to create automated, multi-step customer journeys triggered by real-time behavioral events (such as user registrations, cart abandonments, or purchase completions).

### Key Architectural Concepts

1. **Journeys & Drafts**:
   - A **Journey** represents an automated multi-step communication and decision workflow.
   - Journeys exist in **Draft** state while under design and become **Active** once published.

2. **Visual Graph & Canvas Nodes**:
   - Workflows are constructed visually on a drag-and-drop canvas using nodes connected by edges.
   - All standard canvas nodes adhere to standardized dimensions (**346px width × 112px height**) for consistent layout alignment and visual clarity.

3. **Canonical Intermediate Representation (IR)**:
   - When a draft is published, the engine compiles the visual graph into an immutable **Canonical IR** graph schema.
   - Every revision produces a versioned hash checksum to guarantee execution determinism.

4. **Workflow Execution Runs & Test Lanes**:
   - Published journeys execute as distributed, stateful workflows.
   - **Production Runs** process live customer event streams.
   - **Test Runs** process synthetic or static audience lists for pre-launch validation.

5. **Catalogs & Static Audience Lists**:
   - **Catalogs** store reusable event schemas, action templates (Email, SMS, Push, In-App, Webhook), global parameters, and conversion metrics.
   - **Static Audience Lists** store CSV customer cohorts for automated testing and batch segment validation.

---

## Quick Start Guide: Building Your First Journey

Follow these five steps to build, validate, test, and publish a new customer journey:

### Step 1: Create a New Journey Draft
1. Navigate to the **Journeys** page (\`#journeys\`) from the top navigation bar.
2. Click the **+ New Journey** button in the upper right.
3. In the **Create Journey Modal**, enter a descriptive **Journey Name**, an optional **Description**, and select the **Tenant ID**.
4. Click **Create Draft**. The app will automatically navigate to the visual **Canvas Editor**.

### Step 2: Assemble Nodes on the Canvas
1. Drag node components from the **Palette** on the left onto the workspace canvas:
   - **Event Start Node**: Select the trigger event (e.g., \`user_signup\`).
   - **Condition Node**: Add logic rules to branch based on customer attributes (e.g., \`user.is_premium == true\`).
   - **Experiment Node**: Add multi-variant splits to test message performance.
   - **Action Nodes**: Add communication channels (Email, Push, SMS, In-App, or Webhook).
   - **Exit Node**: Terminate the workflow.
2. Connect nodes by dragging wires between output handles and input handles.

### Step 3: Configure Node Properties
1. Click on any canvas node to open the **Node Inspector** on the right.
2. Set node parameters, template subject lines, delay durations, or variant split percentages.
3. Add custom key-value tokens using the **Parameter Modal**.

### Step 4: Validate and Test Run
1. Click **Validate Graph** on the Canvas Toolbar to ensure there are no disconnected handles or missing parameters.
2. Click **Auto Layout** to organize the graph neatly.
3. Click **Test Run** to trigger a test execution using a sample **Subject ID** or **Static Audience List**.
4. Verify execution step behavior in the **Run Detail** view (\`#run-detail\`).

### Step 5: Publish Version
1. Click **Publish** on the Canvas Toolbar.
2. The engine will compile the draft into a published version and generate a immutable IR revision checksum.
3. Track version history and diffs anytime on the **Version History** page (\`#history\`).
`;

const pagesMarkdown = `# User Guide: Application Pages & Views

This guide provides detailed documentation for every page and view in the Event-Driven Customer Journey Engine application.

---

## Page Index

1. [Journeys Management Page (\`#journeys\`)](#1-journeys-management-page)
2. [Visual Journey Canvas Editor (\`#canvas\`)](#2-visual-journey-canvas-editor)
3. [Catalog Directory Page (\`#catalog\`)](#3-catalog-directory-page)
4. [Run History & Execution Monitoring Page (\`#runs\`)](#4-run-history--execution-monitoring-page)
5. [Run Detail & Execution Trace Inspector (\`#run-detail\`)](#5-run-detail--execution-trace-inspector)
6. [Version History & Canonical IR Diffs Page (\`#history\`)](#6-version-history--canonical-ir-diffs-page)
7. [Static Audience Lists Page (\`#static-lists\`)](#7-static-audience-lists-page)

---

## 1. Journeys Management Page

### Purpose
The **Journeys Management Page** serves as the central directory for all journey workflow drafts and published workflows across tenants.

### Key UI Elements & Controls
- **Page Header**: Displays count of active journey drafts and quick statistics.
- **Search Input**: Text filter matching journey names, descriptions, or draft IDs.
- **Status Filter Dropdown**: Filter by lifecycle status: \`All\`, \`Draft\`, \`Active\`, \`Paused\`, \`Archived\`.
- **Paginated Journeys Table**: Columns for Name, Status, Version, Node Count, Tenant ID, Updated At, and Actions.

---

## 2. Visual Journey Canvas Editor

### Purpose
The **Visual Journey Canvas Editor** is the interactive graph design environment where users visually construct, validate, simulate, test run, and publish customer journeys.

### Key UI Elements & Controls
- **Canvas Toolbar**: Actions for Save, Auto Layout, Validate, Test Run, Publish, and Zoom Controls.
- **Node Palette Drawer**: Left-hand sidebar containing drag-and-drop trigger, decision, timing, and action nodes.
- **Node Inspector Drawer**: Right-hand panel for configuring parameters of the selected canvas node.
- **MiniMap & Viewport**: Canvas navigation preview showing node locations.

---

## 3. Catalog Directory Page

### Purpose
The **Catalog Directory Page** manages reusable event schemas, channel templates (Email, SMS, Push, In-App, Webhook), global parameters, attributes, and conversion metrics.

### Key UI Elements & Controls
- **Category Tabs**: Filter by \`Events\`, \`Actions\`, \`Attributes\`, \`Parameters\`, \`Metrics\`, and \`Templates\`.
- **Create Component Button**: Opens the creation modal for adding reusable catalog definitions.
`;

const componentsMarkdown = `# User Guide: UI Components, Canvas Nodes & Modals Reference

This reference document provides specifications for all UI components, canvas nodes, inspector panels, modals, and design standards.

---

## 1. Canvas Node Dimensions & Design Contract

All standard workflow canvas nodes adhere to a strict dimensional contract:

| Property | Standard Value | Description |
| :--- | :--- | :--- |
| **Node Width** | \`346px\` | Fixed width across all node types. |
| **Node Height** | \`112px\` | Fixed height across all node types. |
| **Box Sizing** | \`border-box\` | Padding and borders calculated within fixed boundaries. |
| **Border Radius** | \`8px\` (\`rounded-lg\`) | Consistent corner curvature across node containers. |
| **Background Color** | Dark Slate (\`#171B26\` / \`#0F131D\`) | Dark-theme container background with backdrop blur. |

---

## 2. Workflow Node Types & Configurations

- **Event Start Node**: Workflow trigger node listening for customer behavioral events.
- **Condition Node**: Branching node evaluating boolean rules or subject attribute expressions.
- **Experiment Node**: Multi-variant split node allocating percentage traffic across variants.
- **Delay Node**: Timer node pausing workflow execution for configured duration (seconds, minutes, hours, days).
- **Wait For Event Node**: Signal gate waiting for a specific event with a timeout fallback.
- **Action Nodes**: Email, SMS, Mobile Push, In-App Message, and Webhook dispatch nodes.
- **Exit Node**: Workflow termination node marking execution exit.

---

## 3. Application Modals & Dialogs

- **Create Journey Modal**: Form for establishing journey title, description, and tenant metadata.
- **Catalog Creation Modal**: Form for registering event schemas, channel templates, parameters, and attributes.
- **Test Run Modal**: Execution launcher for selecting static audience lists, subject IDs, and provider failure modes.
- **Conflict Dialog**: Concurrent edit resolution dialog comparing local ETag hash against server revision.
- **Parameter Modal**: Token selection picker for inserting variables into message body templates.
`;

export type DocsTab = 'overview' | 'pages' | 'components';

const DEFAULT_DOCS_PARAMS = {
  tab: 'overview',
  search: '',
};

function filterAndHighlightMarkdown(markdown: string, search: string): string {
  const query = search.trim();
  if (!query) return markdown;

  const qLower = query.toLowerCase();
  const sections = markdown.split(/(?=\n## |\n### |\n---)/g);
  const matchedSections = sections.filter((sec) => sec.toLowerCase().includes(qLower));

  if (matchedSections.length === 0) {
    return `<div className="p-4 bg-amber-500/10 border border-amber-500/30 text-amber-300 text-xs">No documentation sections found matching "<strong>${query}</strong>". Try searching for <em>canvas</em>, <em>nodes</em>, <em>runs</em>, or <em>catalog</em>.</div>`;
  }

  return matchedSections.join('\n\n');
}

export const DocumentationPage: React.FC = () => {
  const [params, setParams] = useRouteParams(DEFAULT_DOCS_PARAMS);

  const activeTab = (params.tab as DocsTab) || 'overview';

  const rawMarkdown = useMemo(() => {
    switch (activeTab) {
      case 'pages':
        return pagesMarkdown;
      case 'components':
        return componentsMarkdown;
      case 'overview':
      default:
        return overviewMarkdown;
    }
  }, [activeTab]);

  const htmlContent = useMemo(() => {
    try {
      const filteredMarkdown = filterAndHighlightMarkdown(rawMarkdown, params.search || '');
      let parsedHtml = marked.parse(filteredMarkdown) as string;

      if (params.search && params.search.trim()) {
        const term = params.search.trim().replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        const regex = new RegExp(`(${term})`, 'gi');
        parsedHtml = parsedHtml.replace(/>([^<]+)</g, (_match, group1) => {
          return '>' + group1.replace(regex, '<mark class="bg-[#b76dff]/30 text-[#ddb7ff] px-1 font-semibold">$1</mark>') + '<';
        });
      }

      return parsedHtml;
    } catch {
      return '<p class="text-rose-400">Failed to render documentation content.</p>';
    }
  }, [rawMarkdown, params.search]);
  const tabs: { key: DocsTab; label: string; icon: string }[] = [
    { key: 'overview', label: 'Overview & Quick Start', icon: 'menu_book' },
    { key: 'pages', label: 'Application Views Guide', icon: 'auto_stories' },
    { key: 'components', label: 'Nodes & Components Specs', icon: 'widgets' },
  ];

  return (
    <DirectoryLayout
      title="User Documentation & Architecture Guide"
      subtitle="Interactive, searchable reference for application workflows, visual canvas nodes, and platform operations."
      icon="menu_book"
      iconAccentColor="#ddb7ff"
      controls={
        <div className="w-full space-y-4">
          {/* Section Navigation Bar */}
          <div className="w-full flex items-center border-b border-[#464554] overflow-x-auto">
            {tabs.map((tab) => {
              const isActive = activeTab === tab.key;
              return (
                <button
                  key={tab.key}
                  onClick={() => setParams({ tab: tab.key, search: '' })}
                  style={{
                    backgroundColor: isActive ? 'rgba(183, 109, 255, 0.2)' : 'transparent',
                    color: isActive ? '#ddb7ff' : '#908fa0',
                    border: 'none',
                    borderBottom: isActive ? '2px solid #ddb7ff' : '2px solid transparent',
                    borderRadius: '0px',
                    outline: 'none',
                    boxShadow: 'none',
                  }}
                  className={`px-4 py-2.5 text-xs font-semibold flex items-center gap-2 transition-all cursor-pointer whitespace-nowrap rounded-none ${
                    isActive ? 'text-[#ddb7ff]' : 'text-[#908fa0] hover:text-[#dfe2f1]'
                  }`}
                  data-testid={`docs-tab-${tab.key}`}
                >
                  <span className={`material-symbols-outlined text-base ${isActive ? 'text-[#ddb7ff]' : 'text-[#908fa0]'}`}>
                    {tab.icon}
                  </span>
                  <span>{tab.label}</span>
                </button>
              );
            })}
          </div>

          {/* Search Filter Bar */}
          <div className="w-full">
            <input
              type="text"
              placeholder="Search documentation sections..."
              value={params.search}
              onChange={(e) => setParams({ search: e.target.value })}
              data-testid="docs-search-input"
              className="w-full px-3 py-1.5 bg-[#171b26] border border-[#464554] text-white text-xs placeholder-[#908fa0] focus:outline-none focus:border-[#ddb7ff] transition-all rounded-none font-['Outfit',sans-serif]"
            />
          </div>
        </div>
      }
    >
      {/* HTML Rendered Documentation Card */}
      <div className="bg-[#0F131D]/90 backdrop-blur-xl p-8 border-none shadow-xl rounded-none w-full min-h-[500px]">
        <div
          data-testid="rendered-docs-content"
          className="docs-markdown-theme max-w-5xl"
          dangerouslySetInnerHTML={{ __html: htmlContent }}
        />
      </div>
    </DirectoryLayout>
  );
};

export default DocumentationPage;
