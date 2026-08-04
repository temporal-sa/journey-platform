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

async function testNavCycle() {
  console.log('🚀 Testing Edits -> Navigation Cycle in Chrome...');
  const chromePath = getChromePath();

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  const errors = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  page.on('pageerror', (err) => errors.push(err.toString()));

  try {
    await page.goto('http://localhost:3002', { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('#root', { timeout: 5000 });
    await new Promise((r) => setTimeout(r, 1500));

    console.log('👆 Opening Journey Draft in Canvas...');
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
    console.log('✅ Canvas loaded.');

    // Edit a node on the canvas
    console.log('✏️ Making an edit (moving Node 1)...');
    await page.evaluate(() => {
      const node = document.querySelector('.react-flow__node');
      if (node) {
        const r = node.getBoundingClientRect();
        node.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, clientX: r.left + 50, clientY: r.top + 30 }));
        window.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, cancelable: true, clientX: r.left + 150, clientY: r.top + 100 }));
        window.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, clientX: r.left + 150, clientY: r.top + 100 }));
      }
    });

    await new Promise((r) => setTimeout(r, 500));

    // Cycle navigation between pages 3 times
    const routes = ['Journeys Directory', 'Component Catalog', 'Version History', 'Execution Runs', 'Journey Canvas'];
    for (let cycle = 1; cycle <= 3; cycle++) {
      console.log(`🔄 Navigation Cycle ${cycle}...`);
      for (const routeLabel of routes) {
        const btn = await page.$(`button[aria-label="${routeLabel}"]`);
        if (btn) {
          await btn.click();
          await new Promise((r) => setTimeout(r, 400));
        }
      }
    }

    await page.waitForSelector('.react-flow__node', { timeout: 5000 });
    console.log('✅ Returned to Journey Canvas after multiple cycles!');

    const hasErrorText = await page.evaluate(() => {
      return document.body.textContent?.includes('Something went wrong') || document.body.textContent?.includes('Maximum update depth exceeded');
    });

    if (errors.length === 0 && !hasErrorText) {
      console.log('🎉 SUCCESS: Navigated back and forth after canvas edits with zero errors or infinite loops!');
    } else {
      console.error('❌ FAILURE: Errors detected:', errors);
      process.exitCode = 1;
    }

  } catch (err) {
    console.error('❌ Error during nav cycle test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

testNavCycle();
