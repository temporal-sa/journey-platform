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

async function testPaletteScroll() {
  console.log('🚀 Testing Node Palette Scrollability in Chrome...');
  const chromePath = getChromePath();

  const browser = await puppeteer.launch({
    executablePath: chromePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,700'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 700 });

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

    // Inspect Palette Scroll Container
    const scrollInfo = await page.evaluate(() => {
      const drawer = document.querySelector('[data-testid="palette-drawer"]');
      if (!drawer) return null;
      const scrollEl = drawer.querySelector('.overflow-y-auto');
      if (!scrollEl) return null;

      const beforeScroll = scrollEl.scrollTop;
      scrollEl.scrollTop = 150;
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

    console.log('📊 Palette Scroll Diagnostics:', JSON.stringify(scrollInfo, null, 2));

    if (scrollInfo && scrollInfo.overflowYComputed === 'auto' && scrollInfo.afterScroll > 0) {
      console.log('✅ SUCCESS: Node Palette drawer is scrollable and overflow-y: auto is active!');
    } else {
      console.error('❌ FAILURE: Node Palette drawer is not scrollable.');
      process.exitCode = 1;
    }

  } catch (err) {
    console.error('❌ Error during scroll test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

testPaletteScroll();
