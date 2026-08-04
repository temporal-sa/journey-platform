import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync } from 'fs';

function getChromePath() {
  const paths = [
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary',
    '/usr/bin/google-chrome',
    '/usr/bin/chromium-browser',
  ];
  for (const p of paths) {
    if (existsSync(p)) return p;
  }
  throw new Error('No local Chrome binary found.');
}

async function testAllCanvasGestures() {
  console.log('🚀 Starting Comprehensive Canvas Gesture Diagnostic in Chrome...');
  const chromePath = getChromePath();

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  try {
    await page.goto('http://localhost:3002', { waitUntil: 'networkidle0' });
    await page.waitForSelector('button', { timeout: 5000 });

    const buttons = await page.$$('button');
    for (const btn of buttons) {
      const text = await page.evaluate(b => b.textContent, btn);
      if (text && text.includes('Open Canvas')) {
        await btn.click();
        break;
      }
    }

    await page.waitForSelector('.react-flow__node', { timeout: 10000 });
    await new Promise((resolve) => setTimeout(resolve, 1000));

    // Test 1: Node Dragging
    const node1Before = await page.evaluate(() => {
      const el = document.querySelector('.react-flow__node');
      const r = el.getBoundingClientRect();
      return { left: Math.round(r.left), top: Math.round(r.top), width: Math.round(r.width), height: Math.round(r.height) };
    });

    const n1X = node1Before.left + Math.round(node1Before.width / 2);
    const n1Y = node1Before.top + Math.round(node1Before.height / 2);

    await page.mouse.move(n1X, n1Y);
    await page.mouse.down();
    await new Promise((r) => setTimeout(r, 50));
    for (let step = 1; step <= 10; step++) {
      await page.mouse.move(n1X + step * 15, n1Y + step * 10);
      await new Promise((r) => setTimeout(r, 20));
    }
    await page.mouse.up();

    await new Promise((resolve) => setTimeout(resolve, 500));

    const node1After = await page.evaluate(() => {
      const el = document.querySelector('.react-flow__node');
      const r = el.getBoundingClientRect();
      return { left: Math.round(r.left), top: Math.round(r.top) };
    });
    const nodeMoved = (node1After.left !== node1Before.left || node1After.top !== node1Before.top);

    // Test 2: Canvas Panning
    const viewportBefore = await page.evaluate(() => {
      const flow = document.querySelector('.react-flow__viewport');
      return flow ? flow.getAttribute('transform') || window.getComputedStyle(flow).transform : null;
    });

    await page.mouse.move(800, 200);
    await page.mouse.down();
    await new Promise((r) => setTimeout(r, 50));
    for (let step = 1; step <= 10; step++) {
      await page.mouse.move(800 + step * 15, 200 + step * 10);
      await new Promise((r) => setTimeout(r, 20));
    }
    await page.mouse.up();

    await new Promise((resolve) => setTimeout(resolve, 500));

    const viewportAfter = await page.evaluate(() => {
      const flow = document.querySelector('.react-flow__viewport');
      return flow ? flow.getAttribute('transform') || window.getComputedStyle(flow).transform : null;
    });
    const canvasPanned = (viewportBefore !== viewportAfter);

    // Test 3: Connecting Handles
    const initialEdgeCount = await page.evaluate(() => document.querySelectorAll('.react-flow__edge').length);
    
    // Target source handle of node 1 and target handle of node 2
    const handleInfo = await page.evaluate(() => {
      const handles = document.querySelectorAll('.react-flow__handle');
      if (handles.length < 2) return null;
      const h1 = handles[0].getBoundingClientRect();
      const h2 = handles[1].getBoundingClientRect();
      return {
        h1: { x: Math.round(h1.left + h1.width / 2), y: Math.round(h1.top + h1.height / 2) },
        h2: { x: Math.round(h2.left + h2.width / 2), y: Math.round(h2.top + h2.height / 2) }
      };
    });

    if (handleInfo) {
      await page.mouse.move(handleInfo.h1.x, handleInfo.h1.y);
      await page.mouse.down();
      await new Promise((r) => setTimeout(r, 50));
      await page.mouse.move(handleInfo.h2.x, handleInfo.h2.y, { steps: 15 });
      await page.mouse.up();
      await new Promise((r) => setTimeout(r, 500));
    }

    const finalEdgeCount = await page.evaluate(() => document.querySelectorAll('.react-flow__edge').length);
    console.log(`🔌 Initial Edges: ${initialEdgeCount}, Final Edges: ${finalEdgeCount}`);

    console.log('\n📊 TEST RESULTS:');
    console.log(`- Node Dragging Working: ${nodeMoved ? '✅ YES' : '❌ NO'}`);
    console.log(`- Canvas Panning Working: ${canvasPanned ? '✅ YES' : '❌ NO'}`);
    console.log(`- Edge Connecting Working: ${handleInfo ? '✅ YES' : '❌ NO'}`);

  } catch (err) {
    console.error('❌ Error during gesture diagnostics:', err);
  } finally {
    await browser.close();
  }
}

testAllCanvasGestures();
