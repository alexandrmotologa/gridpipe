const fs = require('fs');
const path = require('path');
const puppeteer = require('puppeteer-core');
const { execSync } = require('child_process');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const URL = 'http://localhost:7575';
const IMAGES_DIR = path.resolve(__dirname, '../docs/images');

const FIXTURES = {
  invoice: path.resolve(__dirname, '../examples/messy_sheets/messy_supplier_invoice.xlsx'),
  pnl: path.resolve(__dirname, '../examples/messy_sheets/multi_tier_pnl.xlsx'),
  formulas: path.resolve(__dirname, '../examples/messy_sheets/broken_formulas.xlsx'),
};

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function run() {
  console.log('Starting high-resolution UI capture with Chrome...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-gpu'],
    defaultViewport: {
      width: 1440,
      height: 900,
      deviceScaleFactor: 2,
    },
  });

  const page = await browser.newPage();

  // -------------------------------------------------------------
  // 1. CAPTURE GRID CANVAS (messy_supplier_invoice.xlsx)
  // -------------------------------------------------------------
  console.log('[1/4] Loading Sheet Studio and uploading supplier invoice...');
  await page.goto(URL, { waitUntil: 'domcontentloaded' });
  await sleep(1500);

  const fileInput = await page.$('input[type="file"]');
  await fileInput.uploadFile(FIXTURES.invoice);
  await sleep(2000);

  console.log('Triggering normalization...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const cleanBtn = btns.find((b) => b.textContent && b.textContent.includes('Normalize'));
    if (cleanBtn) cleanBtn.click();
  });
  await sleep(2500);

  // Click on a cell in the grid to display active formula / cell coordinate in formula bar
  await page.evaluate(() => {
    const cells = Array.from(document.querySelectorAll('tbody td'));
    if (cells.length > 5) {
      cells[5].dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }
  });
  await sleep(500);

  const gridPath = path.join(IMAGES_DIR, 'gridpipe_grid.png');
  await page.screenshot({ path: gridPath, type: 'png' });
  console.log('✓ Captured:', gridPath);

  // -------------------------------------------------------------
  // 2. CAPTURE RECONCILIATION HEATMAP (multi_tier_pnl.xlsx)
  // -------------------------------------------------------------
  console.log('[2/4] Uploading P&L for cross-tab reconciliation...');
  await fileInput.uploadFile(FIXTURES.pnl);
  await sleep(2000);

  // Switch to Reconciliation tab
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('header button'));
    const recTab = btns.find((b) => b.title === 'Reconciliation' || b.textContent.includes('Reconciliation'));
    if (recTab) recTab.click();
  });
  await sleep(1000);

  // Run Reconciliation
  console.log('Running reconciliation engine...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('button'));
    const runBtn = btns.find((b) => b.textContent && b.textContent.includes('Run Reconciliation'));
    if (runBtn) runBtn.click();
  });
  await sleep(2500);

  const reconcilePath = path.join(IMAGES_DIR, 'gridpipe_reconcile.png');
  await page.screenshot({ path: reconcilePath, type: 'png' });
  console.log('✓ Captured:', reconcilePath);

  // -------------------------------------------------------------
  // 3. CAPTURE DATA QUALITY & COMPLIANCE SCORECARD
  // -------------------------------------------------------------
  console.log('[3/4] Navigating to Quality & Audit tab...');
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('header button'));
    const auditTab = btns.find((b) => b.title === 'Quality & Audit' || b.textContent.includes('Quality & Audit'));
    if (auditTab) auditTab.click();
  });
  await sleep(2000);

  const auditPath = path.join(IMAGES_DIR, 'gridpipe_audit.png');
  await page.screenshot({ path: auditPath, type: 'png' });
  console.log('✓ Captured:', auditPath);

  // -------------------------------------------------------------
  // 4. CAPTURE FORMULA DAG & CYCLE DETECTOR (broken_formulas.xlsx)
  // -------------------------------------------------------------
  console.log('[4/4] Uploading formula workbook to inspect calculation DAG...');
  await fileInput.uploadFile(FIXTURES.formulas);
  await sleep(2000);

  // Switch to Formula DAG tab
  await page.evaluate(() => {
    const btns = Array.from(document.querySelectorAll('header button'));
    const dagTab = btns.find((b) => b.title === 'Formula DAG' || b.textContent.includes('Formula DAG'));
    if (dagTab) dagTab.click();
  });
  await sleep(2000);

  // Select the first node in the topological order
  await page.evaluate(() => {
    const nodeItems = Array.from(document.querySelectorAll('div[class*="cursor-pointer"]'));
    if (nodeItems.length > 0) {
      nodeItems[0].dispatchEvent(new MouseEvent('click', { bubbles: true }));
    }
  });
  await sleep(500);

  const dagPath = path.join(IMAGES_DIR, 'gridpipe_dag.png');
  await page.screenshot({ path: dagPath, type: 'png' });
  console.log('✓ Captured:', dagPath);

  await browser.close();
  console.log('✓ All 4 UI screenshots captured!');

  // -------------------------------------------------------------
  // 5. ASSEMBLE ANIMATED DEMO GIF VIA FFMPEG
  // -------------------------------------------------------------
  console.log('Assembling animated demo GIF with FFmpeg...');
  const gifPath = path.join(IMAGES_DIR, 'gridpipe_demo.gif');

  // Create a concat demuxer file with 2.5 seconds per slide
  const concatFile = path.join(IMAGES_DIR, 'slides.txt');
  const slidesContent = [
    `file 'gridpipe_grid.png'`,
    `duration 2.5`,
    `file 'gridpipe_reconcile.png'`,
    `duration 2.5`,
    `file 'gridpipe_audit.png'`,
    `duration 2.5`,
    `file 'gridpipe_dag.png'`,
    `duration 2.5`,
    `file 'gridpipe_dag.png'`, // ffmpeg concat demuxer requirement for last slide
  ].join('\n');

  fs.writeFileSync(concatFile, slidesContent, 'utf8');

  // Run ffmpeg with high-quality palette generation and scaling to 1000px wide
  const ffmpegCmd = `ffmpeg -y -f concat -safe 0 -i "${concatFile}" -vf "fps=10,scale=1000:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128:reserve_transparent=0[p];[s1][p]paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`;
  
  try {
    execSync(ffmpegCmd, { stdio: 'inherit', cwd: IMAGES_DIR });
    console.log('✓ Generated animated demo GIF at:', gifPath);
  } catch (e) {
    console.error('FFmpeg error:', e);
  } finally {
    if (fs.existsSync(concatFile)) fs.unlinkSync(concatFile);
  }
}

run().catch((err) => {
  console.error('Error during run:', err);
  process.exit(1);
});
