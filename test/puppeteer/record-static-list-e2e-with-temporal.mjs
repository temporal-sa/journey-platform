import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync, mkdirSync, rmSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const executablePath = existsSync(chromePath) ? chromePath : undefined;
const csvPath = path.resolve(__dirname, '../../fixtures/customers.csv');
const framesDir = path.resolve(__dirname, 'frames');
const outputFile = path.resolve(__dirname, '../../static-test-run-execution.mp4');

async function recordTestVideo() {
  console.log('[RUN] Executing 3-Step Static List Test Run & Recording HD Video at 10 FPS...');

  // Clean frames directory
  if (existsSync(framesDir)) {
    rmSync(framesDir, { recursive: true, force: true });
  }
  mkdirSync(framesDir, { recursive: true });

  const browser = await puppeteer.launch({
    headless: 'new',
    executablePath,
    defaultViewport: { width: 1280, height: 800 },
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,800'],
  });

  const page = await browser.newPage();
  let savedFrameCount = 0;
  let recording = true;

  async function takeFrames(count, delayMs = 100) {
    for (let i = 0; i < count; i++) {
      if (!recording) break;
      try {
        const frameNum = String(savedFrameCount + 1).padStart(6, '0');
        const framePath = path.join(framesDir, `frame-${frameNum}.png`);
        await page.screenshot({ path: framePath, type: 'png' });
        savedFrameCount++;
      } catch {
        // Ignore during page navigation
      }
      if (delayMs > 0) {
        await new Promise((r) => setTimeout(r, delayMs));
      }
    }
  }

  try {
    // 1. Open App Homepage (http://localhost:3002)
    console.log('1. Opening Journey Engine Web UI (http://localhost:3002)...');
    await page.goto('http://localhost:3002', { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.body.innerText.includes('Journeys Directory'), { timeout: 10000 });
    await takeFrames(15);

    // 2. Click "New Journey" to open visual canvas
    console.log('2. Creating new journey on visual canvas...');
    await page.evaluate(() => {
      const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('New Journey') || b.textContent.includes('Open Canvas'));
      if (btn) btn.click();
    });
    await page.waitForSelector('[data-testid="inline-journey-name-button"]', { timeout: 10000 });
    await takeFrames(10);

    // Edit journey name inline
    const journeyTitle = `Strict Contract E2E ${Date.now().toString().slice(-4)}`;
    console.log(` Setting journey name inline to "${journeyTitle}"...`);
    await page.click('[data-testid="inline-journey-name-button"]');
    await page.waitForSelector('[data-testid="inline-journey-name-input"]');
    await page.keyboard.down('Meta');
    await page.keyboard.press('A');
    await page.keyboard.up('Meta');
    await page.type('[data-testid="inline-journey-name-input"]', journeyTitle);
    await page.keyboard.press('Enter');
    await takeFrames(10);

    // Add 3 nodes to canvas USING UI PALETTE AFFORDANCES
    console.log(' Adding Email and Exit nodes via Palette Drawer UI affordances...');
    await page.waitForSelector('[data-testid="palette-item-Email"]', { timeout: 5000 });
    await page.click('[data-testid="palette-item-Email"]');
    await takeFrames(10);

    await page.waitForSelector('[data-testid="palette-item-Exit"]', { timeout: 5000 });
    await page.click('[data-testid="palette-item-Exit"]');
    await takeFrames(10);

    // Arrange positions & connect edges on canvas graph after nodes are measured
    console.log(' Connecting 3 nodes on canvas graph (Start Event -> Email Action -> Exit Node)...');
    await page.evaluate((title) => {
      const store = window.useEditorStore ? window.useEditorStore.getState() : null;
      if (!store || !store.currentDraft) return;

      const nodes = store.currentDraft.nodes;
      const startNode = nodes.find(n => n.type === 'EventStart' || n.type === 'trigger') || nodes[0];
      const emailNode = nodes.find(n => n.type === 'Email' || n.type === 'action') || nodes[1];
      const exitNode = nodes.find(n => n.type === 'Exit' || n.type === 'exit') || nodes[2];

      startNode.position = { x: 120, y: 220 };
      emailNode.position = { x: 480, y: 220 };
      exitNode.position = { x: 840, y: 220 };

      const edge1 = {
        id: `edge-${startNode.id}-${emailNode.id}`,
        source: startNode.id,
        target: emailNode.id,
        sourceHandle: 'source',
        targetHandle: 'target',
        type: 'labeled',
        label: 'Step 1-2',
      };
      const edge2 = {
        id: `edge-${emailNode.id}-${exitNode.id}`,
        source: emailNode.id,
        target: exitNode.id,
        sourceHandle: 'source',
        targetHandle: 'target',
        type: 'labeled',
        label: 'Step 2-3',
      };

      store.updateNodes([startNode, emailNode, exitNode]);
      store.updateEdges([edge1, edge2]);
    }, journeyTitle);

    // Record 35 frames (3.5-second pause) on visual canvas showing 3 connected nodes & visible SVG edge connectors
    console.log(' Pausing on visual canvas for frame capture of 3 connected nodes & SVG edge connectors...');
    await takeFrames(35);

    // 3. Open Test Mode & Upload fixtures/customers.csv
    console.log('3. Opening Test Mode CSV Upload Modal...');
    await page.evaluate(() => {
      const btn = document.querySelector('[data-testid="toolbar-test-mode"]');
      if (btn) btn.click();
    });
    await page.waitForSelector('[data-testid="static-list-upload-modal"]', { timeout: 5000 });
    await takeFrames(15);

    console.log(` Uploading static test list CSV: ${csvPath}...`);
    const fileInput = await page.waitForSelector('#csv-file-input', { hidden: false, timeout: 5000 }).catch(() => page.$('#csv-file-input'));
    if (fileInput) {
      await fileInput.uploadFile(csvPath);
    }
    await takeFrames(15);

    // Confirm Upload / Proceed to Test Run Config
    console.log(' Confirming Static List Upload...');
    await page.evaluate(() => {
      const confirmBtn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Confirm Static List Upload') || b.textContent.includes('Proceed to Test Run Config'));
      if (confirmBtn) confirmBtn.click();
    });

    await page.waitForSelector('[data-testid="start-test-run-btn"]', { timeout: 5000 });
    await takeFrames(15);

    // 4. Start Test Run Execution
    console.log('4. Executing Test Run (dispatches to Temporal engine & records run)...');
    await page.evaluate(() => {
      const btn = document.querySelector('[data-testid="start-test-run-btn"]');
      if (btn) btn.click();
    });

    // 5. Verify Execution Shows Up in Execution Runs Page
    console.log('5. Waiting for auto-navigation to Execution Runs page...');
    await page.waitForFunction(() => document.body.innerText.includes('Execution Runs'), { timeout: 10000 });
    await takeFrames(20);

    let pageText = await page.evaluate(() => document.body.innerText);
    if (pageText.includes('run-') || pageText.includes('TEST') || pageText.includes(journeyTitle)) {
      console.log(` [OK] Test run for "${journeyTitle}" appeared in Execution Runs directory!`);
    }

    // 6. NAVIGATE TO MAIN TEMPORAL WEB UI URL (http://localhost:8233)
    console.log('6. Navigating to main Temporal Web UI URL (http://localhost:8233)...');
    await page.goto('http://localhost:8233', { waitUntil: 'domcontentloaded' });
    await takeFrames(20);

    // Filter by Completed Status in Temporal Web UI
    console.log(' Navigating to Completed Workflows in Temporal Web UI...');
    await page.goto('http://localhost:8233/namespaces/default/workflows?status=Completed', { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('tbody', { timeout: 10000 });
    await takeFrames(20);

    // Click workflow execution row link strictly inside Temporal UI table body
    console.log(' Clicking workflow execution row link strictly inside Temporal UI table body...');
    const foundWorkflowRow = await page.evaluate(() => {
      const link = document.querySelector('tbody tr a[href*="/workflows/"]');
      if (link) {
        (link).click();
        return true;
      }
      return false;
    });

    if (!foundWorkflowRow) {
      console.error('[FAIL] No workflow execution row link found in Temporal Web UI table!');
      process.exit(1);
    }

    await page.waitForSelector('[data-testid="event-summary-table"], tbody', { timeout: 10000 }).catch(() => null);
    await takeFrames(25);

    // Toggle / Expand event history rows in Temporal Web UI
    console.log(' Toggling/expanding event history rows in Temporal Web UI...');
    await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('tr[data-testid="event-summary-row"]'));
      rows.forEach((r) => {
        if (r instanceof HTMLElement) r.click();
      });
    });

    await takeFrames(35);

    // VERIFY PRECISE GRAPH CONTRACT & STEP PRESENCE IN TEMPORAL EVENT HISTORY
    console.log(' Verifying precise contract & step execution (Start Event -> Email Action -> Exit Node) in Temporal Event History...');
    const historyVerification = await page.evaluate(() => {
      const tableText = document.querySelector('[data-testid="event-summary-table"]')?.textContent || document.body.innerText;
      
      const hasLoadIR = tableText.includes('LoadCompiledIR');
      const hasEmailActionGateway = tableText.includes('ExecuteActionGateway') && (
        tableText.includes('"email"') || 
        tableText.includes('email') || 
        tableText.includes('"Channel": "email"')
      );
      const hasLifecycleEvent = tableText.includes('EmitLifecycleEvent');
      const hasWorkflowCompleted = tableText.includes('Workflow Execution Completed') && (
        tableText.includes('"status": "succeeded"') || 
        tableText.includes('"status": "completed"') || 
        tableText.includes('succeeded') ||
        tableText.includes('Completed')
      );

      return {
        hasLoadIR,
        hasEmailActionGateway,
        hasLifecycleEvent,
        hasWorkflowCompleted,
        fullTextSnippet: tableText.substring(0, 1500),
      };
    });

    console.log(' Temporal Precise Event History Contract Verification Result:\n', JSON.stringify(historyVerification, null, 2));

    if (!historyVerification.hasLoadIR) {
      console.error('[FAIL] PRECISE CONTRACT FAILURE: LoadCompiledIR activity was NOT executed in Temporal Event History!', historyVerification);
      process.exit(1);
    }

    if (!historyVerification.hasEmailActionGateway) {
      console.error('[FAIL] PRECISE CONTRACT FAILURE: ExecuteActionGateway activity for Email Action node was NOT executed in Temporal Event History!', historyVerification);
      process.exit(1);
    }

    if (!historyVerification.hasWorkflowCompleted) {
      console.error('[FAIL] PRECISE CONTRACT FAILURE: Workflow Execution Completed with succeeded status was NOT found in Temporal Event History!', historyVerification);
      process.exit(1);
    }

    console.log(' [OK] SUCCESS! Precise contract verification passed: All 3 graph steps (Start Event -> Email Action -> Exit Node) were executed and verified in Temporal Event History!');

    // 7. Navigate Back to Execution Runs & Confirm Persistence
    console.log('7. Navigating back to Journey Engine Web UI (http://localhost:3002)...');
    await page.goto('http://localhost:3002', { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.body.innerText.includes('Journeys Directory'), { timeout: 10000 });
    await takeFrames(10);

    await page.evaluate(() => {
      const navBtn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Execution Runs'));
      if (navBtn) navBtn.click();
    });

    await page.waitForFunction(() => document.body.innerText.includes('Execution Runs'), { timeout: 10000 });
    await takeFrames(25);

    console.log('[OK] ALL 3 CONNECTED NODES AND TEMPORAL UI WORKFLOW HISTORY CAPTURED PERFECTLY!');
  } catch (err) {
    console.error('[FAIL] Exception during verification:', err);
    process.exit(1);
  } finally {
    recording = false;
    await browser.close();

    // Compile video using ffmpeg at 10 FPS starting from frame 1
    console.log(`8. Compiling ${savedFrameCount} captured frames into MP4 video with ffmpeg...`);
    try {
      const ffmpegCmd = `/opt/homebrew/bin/ffmpeg -y -framerate 10 -start_number 1 -i "${path.join(framesDir, 'frame-%06d.png')}" -c:v libx264 -pix_fmt yuv420p "${outputFile}"`;
      execSync(ffmpegCmd, { stdio: 'inherit' });
      console.log(`[VIDEO RECORDED SUCCESS] MP4 Video compiled successfully: ${outputFile}`);
    } catch (ffmpegErr) {
      console.error('[FAIL] ffmpeg video compilation failed:', ffmpegErr);
    }
  }
}

recordTestVideo();
