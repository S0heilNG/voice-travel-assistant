import { chromium } from 'playwright';
import PQueue from 'p-queue';

const MAX_CONCURRENT_SESSIONS = 2;
const TASK_TIMEOUT_MS = 45000;

const queue = new PQueue({ concurrency: MAX_CONCURRENT_SESSIONS });

function withTimeout(promise, ms) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`task timed out after ${ms}ms`)), ms);
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer));
}

export function runSearch(taskFn) {
  return queue.add(async () => {
    const browser = await chromium.launch({ headless: true });
    try {
      const context = await browser.newContext({ viewport: { width: 1366, height: 900 } });
      return await withTimeout(taskFn(context), TASK_TIMEOUT_MS);
    } finally {
      await browser.close();
    }
  });
}
