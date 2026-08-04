import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';

function getChromePath() {
  return '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
}

async function debugMiniMapCSS() {
  const browser = await puppeteer.launch({
    executablePath: getChromePath(),
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

    const cssInfo = await page.evaluate(() => {
      const mm = document.querySelector('.react-flow__minimap');
      if (!mm) return null;
      const cs = window.getComputedStyle(mm);
      return {
        className: mm.className.baseVal || mm.className,
        position: cs.position,
        top: cs.top,
        bottom: cs.bottom,
        left: cs.left,
        right: cs.right,
      };
    });

    console.log('📍 MiniMap CSS positioning:', JSON.stringify(cssInfo, null, 2));

  } catch (err) {
    console.error('❌ Error:', err);
  } finally {
    await browser.close();
  }
}

debugMiniMapCSS();
