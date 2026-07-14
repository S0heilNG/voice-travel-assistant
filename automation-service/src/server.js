import express from 'express';
import { runSearch } from './browserPool.js';

const PORT = process.env.PORT || 4000;

const app = express();
app.use(express.json());

app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

async function handleSearch(url, req, res) {
  const { origin, destination, date } = req.body || {};
  console.log(`search request: url=${url} origin=${origin} destination=${destination} date=${date}`);

  const start = Date.now();
  try {
    const title = await runSearch(async (context) => {
      const page = await context.newPage();
      await page.goto(url, { waitUntil: 'load', timeout: 30000 });
      return page.title();
    });
    res.json({ ok: true, title, tookMs: Date.now() - start });
  } catch (err) {
    res.status(500).json({ ok: false, error: err.message });
  }
}

app.post('/search/flights', (req, res) => handleSearch('https://780.ir/tourism/flights', req, res));
app.post('/search/hotels', (req, res) => handleSearch('https://780.ir/tourism/hotel', req, res));

app.listen(PORT, () => {
  console.log(`automation-service listening on :${PORT}`);
});
