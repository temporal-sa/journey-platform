import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';
import { existsSync } from 'fs';

const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const executablePath = existsSync(chromePath) ? chromePath : undefined;

async function inspectTemporalDOM() {
  console.log('[INSPECT] Inspecting Temporal Web UI 2.37.2 Event History DOM structure...');

  const browser = await puppeteer.launch({
    headless: 'new',
    executablePath,
    defaultViewport: { width: 1280, height: 800 },
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1280,800'],
  });

  const page = await browser.newPage();

  try {
    console.log('1. Navigating to Completed Workflows in Temporal Web UI...');
    await page.goto('http://localhost:8233/namespaces/default/workflows?status=Completed', { waitUntil: 'networkidle2' });
    await new Promise((r) => setTimeout(r, 2000));

    console.log('2. Clicking first completed workflow row...');
    const linkClicked = await page.evaluate(() => {
      const link = document.querySelector('tbody tr a[href*="/workflows/"]');
      if (link) {
        link.click();
        return link.getAttribute('href');
      }
      return null;
    });

    console.log(' Clicked Workflow Link:', linkClicked);
    await new Promise((r) => setTimeout(r, 3000));

    console.log(' Current Page URL:', page.url());

    // Inspect Event History DOM
    const historyStructure = await page.evaluate(() => {
      const allButtons = Array.from(document.querySelectorAll('button')).map((b) => ({
        text: b.textContent.trim(),
        ariaLabel: b.getAttribute('aria-label'),
        className: b.className,
      }));

      const allTableHeaders = Array.from(document.querySelectorAll('th')).map((th) => th.textContent.trim());

      const eventRows = Array.from(document.querySelectorAll('tr, [role="row"], .event-row, [data-testid*="event"]')).map((el) => ({
        text: el.textContent.trim(),
        tagName: el.tagName,
        className: el.className,
        dataTestId: el.getAttribute('data-testid'),
      }));

      const bodyText = document.body.innerText;

      return {
        allButtons: allButtons.slice(0, 15),
        allTableHeaders,
        eventRowsCount: eventRows.length,
        sampleEventRows: eventRows.slice(0, 10),
        bodyTextSnippet: bodyText.substring(0, 1000),
      };
    });

    console.log('=== TEMPORAL HISTORY DOM INSPECTION RESULTS ===');
    console.log(JSON.stringify(historyStructure, null, 2));

  } catch (err) {
    console.error('[FAIL] Exception during DOM inspection:', err);
  } finally {
    await browser.close();
  }
}

inspectTemporalDOM();
