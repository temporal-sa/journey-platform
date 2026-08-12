import path from 'path';
import { existsSync, mkdirSync, writeFileSync, rmSync } from 'fs';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';
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

const delay = (ms) => new Promise((res) => setTimeout(res, ms));

async function runExperimentVideoTest() {
  console.log('🚀 Starting Puppeteer Video Recording E2E Test...');
  const executablePath = getChromePath();

  const framesDir = path.resolve(__dirname, 'tmp_frames');
  if (existsSync(framesDir)) {
    rmSync(framesDir, { recursive: true, force: true });
  }
  mkdirSync(framesDir, { recursive: true });

  const browser = await puppeteer.launch({
    executablePath,
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });

  const client = await page.target().createCDPSession();
  const frameBuffers = [];

  client.on('Page.screencastFrame', async (event) => {
    frameBuffers.push(Buffer.from(event.data, 'base64'));
    try {
      await client.send('Page.screencastFrameAck', { sessionId: event.sessionId });
    } catch {}
  });

  await client.send('Page.startScreencast', {
    format: 'png',
    quality: 100,
    everyNthFrame: 1,
  });

  try {
    const frontendUrl = 'http://localhost:3003';
    console.log(`🌐 Navigating to ${frontendUrl}/#/experiments ...`);
    await page.goto(`${frontendUrl}/#/experiments`, { waitUntil: 'networkidle0' });
    await delay(1000);

    // 1. Verify Experiment Analytics dashboard
    await page.waitForSelector('[data-testid="experiment-report-view"]', { timeout: 5000 });
    console.log('✅ Dashboard loaded.');
    await delay(1000);

    // 2. Locate dropdown selector and highlight for visual clarity
    const selectorId = '[data-testid="experiment-selector-dropdown"]';
    await page.waitForSelector(selectorId, { timeout: 5000 });

    await page.evaluate((sel) => {
      const el = document.querySelector(sel);
      if (el) {
        el.style.outline = '3px solid #b76dff';
        el.style.boxShadow = '0 0 15px rgba(183, 109, 255, 0.8)';
      }
    }, selectorId);
    await delay(1000);

    // 3. Select 'exp-checkout-cta-v1'
    console.log('🖱️ Switching dropdown option to "exp-checkout-cta-v1"...');
    await page.select(selectorId, 'exp-checkout-cta-v1');
    await delay(1500);

    // 4. Verify updated title and elements
    await page.waitForSelector('[data-testid="srm-status-card"]', { timeout: 3000 });
    console.log('✅ Metrics and SRM Card loaded for exp-checkout-cta-v1.');
    await delay(2000);

    console.log('⏹️ Stopping Screencast...');
    await client.send('Page.stopScreencast');

    console.log(`📸 Saving ${frameBuffers.length} recorded frames...`);
    frameBuffers.forEach((buf, idx) => {
      const frameNum = String(idx + 1).padStart(5, '0');
      writeFileSync(path.join(framesDir, `frame_${frameNum}.png`), buf);
    });

    const videoOutputPath = path.resolve(__dirname, 'experiment-flow-test.mp4');
    console.log(`🎥 Encoding video with ffmpeg to ${videoOutputPath} ...`);

    const ffmpegCmd = `/opt/homebrew/bin/ffmpeg -y -framerate 8 -i "${framesDir}/frame_%05d.png" -c:v libx264 -pix_fmt yuv420p -vf "scale=trunc(iw/2)*2:trunc(ih/2)*2" "${videoOutputPath}"`;
    execSync(ffmpegCmd, { stdio: 'inherit' });

    console.log(`🎬 Video recording successfully created at: ${videoOutputPath}`);
  } catch (err) {
    console.error('❌ Video Recording E2E Test Failed:', err);
    process.exit(1);
  } finally {
    await browser.close();
    if (existsSync(framesDir)) {
      rmSync(framesDir, { recursive: true, force: true });
    }
  }
}

runExperimentVideoTest();
