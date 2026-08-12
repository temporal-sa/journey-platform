import path from 'path';
import { existsSync } from 'fs';
import { fileURLToPath } from 'url';
import puppeteer from '../../web/node_modules/puppeteer-core/lib/puppeteer/puppeteer-core.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

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
  throw new Error('No local Chrome binary found. Please install Google Chrome.');
}

async function runExperimentE2ETest() {
  console.log('🚀 Launching Puppeteer E2E Experiment Flow & Dropdown Test...');
  const executablePath = getChromePath();
  console.log(`📍 Chrome Binary: ${executablePath}`);

  const browser = await puppeteer.launch({
    executablePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  try {
    const frontendUrl = 'http://localhost:3003';
    console.log(`🌐 Navigating to ${frontendUrl}/#/experiments ...`);
    await page.goto(`${frontendUrl}/#/experiments`, { waitUntil: 'networkidle0' });

    // 1. Verify Experiment Analytics dashboard rendered
    await page.waitForSelector('[data-testid="experiment-report-view"]', { timeout: 5000 });
    console.log('✅ Experiment Analytics Report Dashboard loaded successfully.');

    // 2. Verify Dropdown Selector is present and populates available experiments
    const selectorId = '[data-testid="experiment-selector-dropdown"]';
    await page.waitForSelector(selectorId, { timeout: 5000 });
    console.log('✅ Experiment dropdown selector is visible.');

    const options = await page.evaluate((sel) => {
      const select = document.querySelector(sel);
      if (!select) return [];
      return Array.from(select.options).map((opt) => ({
        value: opt.value,
        text: opt.text,
      }));
    }, selectorId);

    console.log(`📋 Discovered ${options.length} experiment(s) in dropdown:`, options);

    if (options.length === 0) {
      throw new Error('Expected at least 1 experiment in dropdown, got 0.');
    }

    const hasCheckoutExp = options.some((opt) => opt.value === 'exp-checkout-cta-v1');
    if (!hasCheckoutExp) {
      throw new Error('Expected "exp-checkout-cta-v1" to be available in dropdown options.');
    }

    // 3. Select 'exp-checkout-cta-v1' from dropdown and verify report updates
    await page.select(selectorId, 'exp-checkout-cta-v1');

    // 4. Verify metrics elements and non-zero values
    await page.waitForSelector('[data-testid="srm-status-card"]', { timeout: 3000 });
    await page.waitForSelector('[data-testid="data-freshness-indicator"]', { timeout: 3000 });

    const metrics = await page.evaluate(() => {
      const getText = (id) => document.querySelector(`[data-testid="${id}"]`)?.textContent || '';
      return {
        assigned: getText('metric-assigned'),
        exposed: getText('metric-exposed'),
        conversions: getText('metric-conversion'),
        conversionRate: getText('metric-conversion-rate'),
        maxLift: getText('metric-max-lift'),
      };
    });

    console.log('📊 Verified Displayed Metrics:', metrics);

    if (metrics.assigned === '0' || metrics.exposed === '0') {
      throw new Error(`Expected non-zero assigned and exposed metrics, got: ${JSON.stringify(metrics)}`);
    }

    console.log('🎉 ALL PUPPETEER EXPERIMENT METRICS & DROPDOWN E2E TESTS PASSED SUCCESSFULLY!');
  } catch (err) {
    console.error('❌ Puppeteer E2E Test Failed:', err);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

runExperimentE2ETest();
