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

async function testInspectorScroll() {
  console.log('🚀 Testing Node Inspector Scrollability in Chrome (500px viewport)...');
  const chromePath = getChromePath();

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,500'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 500 });

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

    console.log('👆 Clicking Email action node to open Inspector panel...');
    const emailNode = await page.$('[data-node-type="Email"]');
    if (emailNode) {
      await emailNode.click();
    } else {
      const anyNode = await page.$('.react-flow__node');
      if (anyNode) await anyNode.click();
    }

    await page.waitForSelector('[data-testid="node-inspector-panel"]', { timeout: 5000 });
    console.log('✅ Inspector panel opened.');

    await new Promise((resolve) => setTimeout(resolve, 500));

    // Inspect Node Inspector Scroll Container
    const scrollInfo = await page.evaluate(() => {
      const panel = document.querySelector('[data-testid="node-inspector-panel"]');
      if (!panel) return null;
      const scrollEl = panel.querySelector('.overflow-y-auto');
      if (!scrollEl) return null;

      const beforeScroll = scrollEl.scrollTop;
      scrollEl.scrollTop = 100;
      const afterScroll = scrollEl.scrollTop;

      const cs = window.getComputedStyle(scrollEl);

      return {
        clientHeight: scrollEl.clientHeight,
        scrollHeight: scrollEl.scrollHeight,
        isScrollable: scrollEl.scrollHeight > scrollEl.clientHeight,
        beforeScroll,
        afterScroll,
        overflowYComputed: cs.overflowY,
      };
    });

    console.log('📊 Inspector Scroll Diagnostics:', JSON.stringify(scrollInfo, null, 2));

    if (scrollInfo && scrollInfo.overflowYComputed === 'auto' && scrollInfo.isScrollable && scrollInfo.afterScroll > 0) {
      console.log('✅ SUCCESS: Node Inspector panel content is vertically scrollable in Chrome!');
    } else {
      console.error('❌ FAILURE: Node Inspector panel is not scrolling as expected.');
      process.exitCode = 1;
    }

  } catch (err) {
    console.error('❌ Error during inspector scroll test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

testInspectorScroll();
