import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync } from 'fs';

function getChromePath() {
  const paths = [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary',
    '/usr/bin/google-chrome',
    '/usr/bin/chromium-browser',
  ];
  for (const path of paths) {
    if (existsSync(path)) return path;
  }
  throw new Error('No local Chrome binary found. Please install Google Chrome.');
}

async function runPaletteAnchoringTest() {
  console.log('🚀 Launching Puppeteer E2E Layout Stability Test...');
  const executablePath = getChromePath();
  console.log(`📍 Chrome Binary: ${executablePath}`);

  const browser = await puppeteer.launch({
    executablePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  const port = process.env.FRONTEND_PORT || '3002';
  try {
    console.log(`🌐 Navigating to http://localhost:${port} ...`);
    await page.goto(`http://localhost:${port}`, { waitUntil: 'domcontentloaded' });
    await new Promise((r) => setTimeout(r, 800));

    // Look for canvas nav button in sidebar
    console.log('👆 Navigating to Journey Canvas route...');
    await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button'));
      const canvasBtn = btns.find((b) => b.textContent?.includes('Journey Canvas'));
      if (canvasBtn) canvasBtn.click();
    });
    await new Promise((r) => setTimeout(r, 800));

    // Wait for Canvas editor container
    await page.waitForSelector('[data-testid="canvas-editor-container"]', { timeout: 5000 });
    console.log('✅ Journey Canvas loaded successfully.');

    // Click palette item to add & select a node
    console.log('👆 Clicking EventStart node in palette to add node...');
    const paletteItem = await page.waitForSelector('[data-testid="palette-item-EventStart"]', { timeout: 3000 });
    await paletteItem.click();
    await new Promise((r) => setTimeout(r, 500));

    // Click node on canvas to open Node Inspector drawer
    const nodeSelector = '.react-flow__node';
    await page.waitForSelector(nodeSelector, { timeout: 3000 });
    await page.click(nodeSelector);
    await page.waitForSelector('[data-testid="inspector-cancel-btn"]', { timeout: 3000 });
    console.log('✅ Node Inspector drawer opened with Discard & Save buttons.');

    // Function to capture element bounds
    const captureLayoutBounds = async () => {
      return await page.evaluate(() => {
        const getBounds = (selector) => {
          const el = document.querySelector(selector);
          if (!el) return null;
          const rect = el.getBoundingClientRect();
          return {
            top: Math.round(rect.top),
            bottom: Math.round(rect.bottom),
            left: Math.round(rect.left),
            right: Math.round(rect.right),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
          };
        };

        return {
          miniMap: getBounds('.react-flow__minimap'),
          canvasToolbar: getBounds('[data-testid="canvas-toolbar"]'),
          discardBtn: getBounds('[data-testid="inspector-cancel-btn"]'),
          saveNodeBtn: getBounds('[data-testid="inspector-save-btn"]'),
        };
      });
    };

    // Step 1: Initial Bounds (Default State)
    const initialBounds = await captureLayoutBounds();
    console.log('\n📊 Initial Element Positions:');
    console.table(initialBounds);

    // Step 2: Click "COLLAPSE ALL" to collapse all palette subpanels
    const toggleAllBtn = await page.waitForSelector('aside[data-testid="palette-drawer"] button');
    await toggleAllBtn.click();
    await new Promise((r) => setTimeout(r, 500));

    const collapsedBounds = await captureLayoutBounds();
    console.log('\n📊 Collapsed Palette Element Positions:');
    console.table(collapsedBounds);

    // Step 3: Click "EXPAND ALL" to expand all palette subpanels
    await toggleAllBtn.click();
    await new Promise((r) => setTimeout(r, 500));

    const expandedBounds = await captureLayoutBounds();
    console.log('\n📊 Expanded Palette Element Positions:');
    console.table(expandedBounds);

    // Assertions: Verify position shift is 0px
    let pass = true;
    const elementsToVerify = ['miniMap', 'canvasToolbar', 'discardBtn', 'saveNodeBtn'];

    console.log('\n🔍 Verifying Layout Shifts (Default vs Collapsed vs Expanded):');
    elementsToVerify.forEach((elem) => {
      const initial = initialBounds[elem];
      const expanded = expandedBounds[elem];

      if (!initial || !expanded) {
        console.error(`❌ Missing element bounds for ${elem}`);
        pass = false;
        return;
      }

      const topShift = Math.abs(initial.top - expanded.top);
      const bottomShift = Math.abs(initial.bottom - expanded.bottom);

      if (topShift === 0 && bottomShift === 0) {
        console.log(`  ✅ [${elem}] Top Shift: ${topShift}px, Bottom Shift: ${bottomShift}px (STABLE)`);
      } else {
        console.error(`  ❌ [${elem}] Top Shift: ${topShift}px, Bottom Shift: ${bottomShift}px (MOVED!)`);
        pass = false;
      }
    });

    if (pass) {
      console.log('\n🎉 TEST PASSED! All UI elements remained completely stationary during palette expansion.');
    } else {
      console.error('\n💥 TEST FAILED! Layout shift detected when expanding palette subpanels.');
      process.exitCode = 1;
    }
  } catch (err) {
    console.error('❌ Error executing Puppeteer test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

runPaletteAnchoringTest();
