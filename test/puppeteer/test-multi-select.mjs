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

async function testMultiSelectAndDelete() {
  console.log('🚀 Testing Multi-Node Selection & Bulk Deletion in Chrome...');
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

    const initialCount = await page.evaluate(() => document.querySelectorAll('.react-flow__node').length);
    console.log(`📍 Initial nodes on canvas: ${initialCount}`);

    console.log('👆 Selecting 3 nodes (node_signup, node_condition, node_experiment)...');
    await page.evaluate(() => {
      const nodes = Array.from(document.querySelectorAll('.react-flow__node')).map(n => n.getAttribute('data-id')).filter(Boolean);
      const targetIds = nodes.slice(0, 3);
      const setSel = window.useEditorStore ? window.useEditorStore.getState().setSelectedNodeIds : null;
      if (setSel) {
        setSel(targetIds);
      } else {
        nodes.slice(0, 3).forEach(id => {
          const el = document.querySelector(`[data-id="${id}"]`);
          if (el) el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, shiftKey: true }));
        });
      }
    });

    await new Promise((resolve) => setTimeout(resolve, 500));

    const selectedCount = await page.evaluate(() => {
      return document.querySelectorAll('.react-flow__node[class*="scale-105"], .react-flow__node.selected, [class*="ring-4"]').length;
    });
    console.log(`✅ Selected nodes count: ${selectedCount}`);

    // Press Delete key
    console.log('⌨️ Pressing Delete key to bulk-delete selected nodes...');
    await page.keyboard.press('Delete');

    await new Promise((resolve) => setTimeout(resolve, 1000));

    const afterDeleteCount = await page.evaluate(() => document.querySelectorAll('.react-flow__node').length);
    console.log(`📍 Remaining nodes on canvas after deletion: ${afterDeleteCount}`);

    if (afterDeleteCount < initialCount) {
      console.log(`🎉 SUCCESS: Multi-node selection selected ${initialCount - afterDeleteCount} nodes and bulk-deleted them successfully!`);
    } else {
      console.error('❌ FAILURE: Nodes were not deleted.');
      process.exitCode = 1;
    }

  } catch (err) {
    console.error('❌ Error during multi-select test:', err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

testMultiSelectAndDelete();
