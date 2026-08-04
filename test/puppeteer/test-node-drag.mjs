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

async function testNodeDragAndPan() {
  console.log('🚀 Testing Node Dragging & Canvas Panning in Chrome...');
  const chromePath = getChromePath();

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  try {
    await page.goto('http://localhost:3002', { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('#root', { timeout: 5000 });
    await new Promise((r) => setTimeout(r, 1000));

    console.log('👆 Clicking journey row...');
    const card = await page.waitForSelector('button, tr, div', { timeout: 5000 });
    const buttons = await page.$$('button');
    for (const btn of buttons) {
      const text = await page.evaluate(b => b.textContent, btn);
      if (text && text.includes('Open Canvas')) {
        await btn.click();
        console.log('✅ Clicked "Open Canvas".');
        break;
      }
    }

    await page.waitForSelector('.react-flow__node', { timeout: 10000 });
    await new Promise((resolve) => setTimeout(resolve, 1000));

    const nodeInfo = await page.evaluate(() => {
      const node = document.querySelector('[data-node-type="Email"]')?.closest('.react-flow__node') || document.querySelector('.react-flow__node');
      const r = node.getBoundingClientRect();
      return { left: Math.round(r.left), top: Math.round(r.top), width: Math.round(r.width), height: Math.round(r.height) };
    });
    console.log('📍 Initial Node Screen Position:', nodeInfo);

    console.log('⚡ Dispatching drag gesture on Node...');
    await page.evaluate(() => {
      const node = document.querySelector('[data-node-type="Email"]')?.closest('.react-flow__node') || document.querySelector('.react-flow__node');
      if (node) {
        const r = node.getBoundingClientRect();
        const startX = r.left + r.width / 2;
        const startY = r.top + r.height / 2;
        
        node.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: startX, clientY: startY, buttons: 1 }));
        window.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, cancelable: true, clientX: startX + 200, clientY: startY + 100, buttons: 1 }));
        window.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: startX + 200, clientY: startY + 100 }));
      }
    });

    await new Promise((resolve) => setTimeout(resolve, 800));

    const afterDragPos = await page.evaluate(() => {
      const node = document.querySelector('[data-node-type="Email"]')?.closest('.react-flow__node') || document.querySelector('.react-flow__node');
      const r = node.getBoundingClientRect();
      return { left: Math.round(r.left), top: Math.round(r.top) };
    });
    console.log('📍 Position after dragging Node:', afterDragPos);

    const nodeMoved = (afterDragPos.left !== nodeInfo.left || afterDragPos.top !== nodeInfo.top);

    if (nodeMoved) {
      console.log(`🎉 SUCCESS: Node moved from (${nodeInfo.left}, ${nodeInfo.top}) to (${afterDragPos.left}, ${afterDragPos.top})! Click-and-drag node movement is working smoothly!`);
    } else {
      console.error('❌ FAILURE: Node did not move on click-and-drag.');
      process.exitCode = 1;
    }

  } catch (err) {
    console.error('❌ Error during drag test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

testNodeDragAndPan();
