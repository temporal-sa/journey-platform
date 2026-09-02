import puppeteer from '../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync, mkdirSync } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const imagesDir = path.resolve(__dirname, '../docs/images');

if (!existsSync(imagesDir)) {
  mkdirSync(imagesDir, { recursive: true });
}

async function captureScreenshots() {
  console.log('[SCREENSHOTS] Launching Chrome to capture demo screenshots...');

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    defaultViewport: { width: 1440, height: 900, deviceScaleFactor: 2 },
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-gpu',
      '--disable-dev-shm-usage',
    ],
  });

  const page = await browser.newPage();

  try {
    // 1. Journey Authoring Canvas
    console.log('[1/3] Navigating to Journey Canvas...');
    await page.goto('http://localhost:3002/#/canvas', { waitUntil: 'networkidle0', timeout: 15000 });
    await new Promise((r) => setTimeout(r, 2000));

    const canvasShotPath = path.join(imagesDir, '01-journey-canvas-authoring.png');
    await page.screenshot({ path: canvasShotPath, fullPage: false });
    console.log(`[OK] Saved: ${canvasShotPath}`);

    // 2. Run Details Page with Trace Graph & Sub-Runs
    console.log('[2/3] Navigating to Run Detail Page...');
    await page.goto('http://localhost:3002/#/run-detail?runId=tr-7cee5775-f6f0-435d-a708-46bda2f89d2b', {
      waitUntil: 'networkidle0',
      timeout: 15000,
    });
    await new Promise((r) => setTimeout(r, 2000));

    const runDetailShotPath = path.join(imagesDir, '02-run-details-trace-graph.png');
    await page.screenshot({ path: runDetailShotPath, fullPage: false });
    console.log(`[OK] Saved: ${runDetailShotPath}`);

    // 3. Temporal Web UI Workflow Trace
    console.log('[3/3] Navigating to Temporal Web UI...');
    await page.goto('http://localhost:8233/namespaces/default/workflows', {
      waitUntil: 'networkidle0',
      timeout: 15000,
    });
    await new Promise((r) => setTimeout(r, 2000));

    // Try clicking on the first workflow row if present to show granular history
    try {
      const rowSelector = 'tbody tr a, [data-testid="workflow-row"]';
      await page.waitForSelector(rowSelector, { timeout: 3000 });
      await page.click(rowSelector);
      await new Promise((r) => setTimeout(r, 2000));
    } catch {
      console.log('Using main Temporal workflow view');
    }

    const temporalShotPath = path.join(imagesDir, '03-temporal-workflow-trace.png');
    await page.screenshot({ path: temporalShotPath, fullPage: false });
    console.log(`[OK] Saved: ${temporalShotPath}`);

    console.log('[SUCCESS] All screenshots captured successfully.');
  } catch (err) {
    console.error('[ERROR] Capturing screenshots failed:', err);
    throw err;
  } finally {
    await browser.close();
  }
}

captureScreenshots();
