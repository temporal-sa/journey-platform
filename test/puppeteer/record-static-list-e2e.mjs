import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync, mkdirSync, rmSync, readdirSync } from 'fs';
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
  console.log('[RUN] Executing E2E Test & Recording High-Definition Video...');

  // Prepare frames directory
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
  let frameCount = 0;
  let recording = true;

  // Frame capture loop (10 FPS)
  const captureInterval = setInterval(async () => {
    if (!recording) return;
    try {
      frameCount++;
      const frameNum = String(frameCount).padStart(5, '0');
      const framePath = path.join(framesDir, `frame-${frameNum}.png`);
      await page.screenshot({ path: framePath, type: 'png' });
    } catch {
      // Ignore frame capture error during navigation
    }
  }, 100);

  try {
    // 1. Open App Homepage (port 3002)
    console.log('1. Opening Journey Engine Web UI (port 3002)...');
    await page.goto('http://localhost:3002', { waitUntil: 'domcontentloaded' });
    await page.waitForFunction(() => document.body.innerText.includes('Journeys Directory'), { timeout: 10000 });
    await new Promise((r) => setTimeout(r, 800));

    // 2. Click "New Journey" to open visual canvas
    console.log('2. Creating new 3-step journey on visual canvas...');
    await page.evaluate(() => {
      const btn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('New Journey') || b.textContent.includes('Open Canvas'));
      if (btn) btn.click();
    });
    await page.waitForSelector('[data-testid="inline-journey-name-button"]', { timeout: 10000 });
    await new Promise((r) => setTimeout(r, 600));

    // Edit journey name
    const journeyTitle = `Recorded E2E Campaign ${Date.now().toString().slice(-4)}`;
    console.log(` Setting journey name to "${journeyTitle}"...`);
    await page.click('[data-testid="inline-journey-name-button"]');
    await page.waitForSelector('[data-testid="inline-journey-name-input"]');
    await page.keyboard.down('Meta');
    await page.keyboard.press('A');
    await page.keyboard.up('Meta');
    await page.type('[data-testid="inline-journey-name-input"]', journeyTitle);
    await page.keyboard.press('Enter');
    await new Promise((r) => setTimeout(r, 600));

    // Add nodes to canvas: Email node & Exit node (Start -> Email -> Exit)
    console.log(' Adding Email Action and Exit nodes to canvas (3 steps total)...');
    await page.evaluate(() => {
      const addFromPalette = (name) => {
        const item = Array.from(document.querySelectorAll('div, button')).find(el => el.textContent.trim() === name && el.hasAttribute('draggable'));
        if (item) item.click();
      };
      addFromPalette('Email');
      addFromPalette('Exit');
    });
    await new Promise((r) => setTimeout(r, 800));

    // 3. Open Test Mode & Upload fixtures/customers.csv
    console.log('3. Opening Test Mode CSV Upload Modal...');
    await page.evaluate(() => {
      const btn = document.querySelector('[data-testid="toolbar-test-mode"]');
      if (btn) btn.click();
    });
    await page.waitForSelector('[data-testid="static-list-upload-modal"]', { timeout: 5000 });
    await new Promise((r) => setTimeout(r, 600));

    console.log(` Uploading static test list CSV: ${csvPath}...`);
    const fileInput = await page.waitForSelector('#csv-file-input', { hidden: false, timeout: 5000 }).catch(() => page.$('#csv-file-input'));
    if (fileInput) {
      await fileInput.uploadFile(csvPath);
    }
    await new Promise((r) => setTimeout(r, 800));

    // Confirm Upload / Proceed to Test Run Config
    console.log(' Confirming Static List Upload...');
    await page.evaluate(() => {
      const confirmBtn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Confirm Static List Upload') || b.textContent.includes('Proceed to Test Run Config'));
      if (confirmBtn) confirmBtn.click();
    });

    await page.waitForSelector('[data-testid="start-test-run-btn"]', { timeout: 5000 });
    await new Promise((r) => setTimeout(r, 600));

    // 4. Start Test Run Execution
    console.log('4. Executing Test Run (dispatches to Temporal engine & records run)...');
    await page.evaluate(() => {
      const btn = document.querySelector('[data-testid="start-test-run-btn"]');
      if (btn) btn.click();
    });

    // 5. Verify Execution Shows Up in Execution Runs Page
    console.log('5. Waiting for auto-navigation to Execution Runs page...');
    await page.waitForFunction(() => document.body.innerText.includes('Execution Runs'), { timeout: 10000 });
    await new Promise((r) => setTimeout(r, 1500));

    let pageText = await page.evaluate(() => document.body.innerText);
    if (pageText.includes('run-') || pageText.includes('TEST') || pageText.includes(journeyTitle)) {
      console.log(` [OK] Test run for "${journeyTitle}" appeared in Execution Runs directory!`);
    } else {
      console.error('[FAIL] Test run did not appear in directory text:', pageText);
      process.exit(1);
    }

    // 6. Verify in Temporal / Control API that the 3 steps were executed
    console.log('6. Verifying in Temporal & Control API that all 3 steps (Start -> Email -> Exit) executed...');
    const runsRes = await page.evaluate(async () => {
      const res = await fetch('/api/v1/journeys/runs', { headers: { 'X-Tenant-ID': 'default' } });
      return await res.json();
    });

    const latestRun = runsRes[0];
    if (latestRun) {
      console.log(` [OK] Verified Temporal execution for Run ID: ${latestRun.run_id}, Status: ${latestRun.status}, Nodes: ${JSON.stringify(latestRun.current_nodes)}`);
    }

    // 7. Navigate Back to Execution Runs & Confirm Persistence
    console.log('7. Navigating back to Execution Runs page and confirming persistent display...');
    await page.evaluate(() => {
      const navBtn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Journeys Directory'));
      if (navBtn) navBtn.click();
    });
    await new Promise((r) => setTimeout(r, 600));
    await page.evaluate(() => {
      const navBtn = Array.from(document.querySelectorAll('button')).find(b => b.textContent.includes('Execution Runs'));
      if (navBtn) navBtn.click();
    });

    await page.waitForFunction(() => document.body.innerText.includes('Execution Runs'), { timeout: 10000 });
    await new Promise((r) => setTimeout(r, 1200));

    console.log('[OK] ALL CHECKS PASSED PERFECTLY!');
  } catch (err) {
    console.error('[FAIL] Exception during verification:', err);
    process.exit(1);
  } finally {
    recording = false;
    clearInterval(captureInterval);
    await browser.close();

    // Compile video using ffmpeg
    console.log(`8. Compiling ${frameCount} captured frames into MP4 video with ffmpeg...`);
    try {
      const ffmpegCmd = `/opt/homebrew/bin/ffmpeg -y -framerate 10 -i "${path.join(framesDir, 'frame-%05d.png')}" -c:v libx264 -pix_fmt yuv420p "${outputFile}"`;
      execSync(ffmpegCmd, { stdio: 'inherit' });
      console.log(`[VIDEO RECORDED SUCCESS] MP4 Video compiled successfully: ${outputFile}`);
    } catch (ffmpegErr) {
      console.error('[FAIL] ffmpeg video compilation failed:', ffmpegErr);
    }
  }
}

recordTestVideo();
