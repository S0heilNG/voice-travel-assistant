# تحلیل تعامل کاربران (نسخه‌ی آزمایشی)

در نسخه‌ی آزمایشی، هر تعامل به‌صورت **ناشناس** در یک دیتابیس SQLite ثبت می‌شود تا
بفهمیم کاربران واقعاً چه می‌خواهند — مخصوصاً چه چیزهایی که دستیار نتوانست کمک کند.

## کجا ذخیره می‌شود

- فایل: `/var/www/vta/shared/analytics.db` (بیرون از پوشه‌ی `releases/`، کنار `.env`).
  چون در `shared/` است، rotation دیپلوی هرگز آن را پاک نمی‌کند.
- مسیر از env `VTA_ANALYTICS_DB` می‌آید (در `shared/.env` ست شده). اگر خالی باشد،
  ثبت کاملاً خاموش است و اپ بدون هیچ مشکلی کار می‌کند.
- هیچ داده‌ی شخصی ثبت نمی‌شود؛ تنها شناسه، یک UUID تصادفی است که فرانت می‌سازد و در
  `sessionStorage` نگه می‌دارد (با بستن تب پاک می‌شود).

## Schema

**`parses`** — یک ردیف به‌ازای هر فراخوانی `/api/parse` (ثبت خودکار در بک‌اند):

| ستون | توضیح |
|---|---|
| `ts` | زمان (UTC, RFC3339) |
| `session_id` | شناسه‌ی تصادفی نشست |
| `raw_text` | متن خام کاربر (حداکثر ۵۰۰ کاراکتر) |
| `intent` | intent تشخیص‌داده‌شده (پرواز داخلی/خارجی، قطار، اتوبوس، هتل، تور، help، unknown) |
| `origin`, `destination` | نام فارسی شهرها (اگر استخراج شد) |
| `date`, `nights` | تاریخ شمسی و تعداد شب (هتل) |
| `missing` | فیلدهای کم (comma-separated) |
| `city_supported` | آیا شهر برای این سرویس پشتیبانی می‌شود |
| `url_from_this_utterance` | آیا **همین یک جمله به‌تنهایی** برای ساخت URL کافی بود (پایین را حتماً بخوانید) |
| `category` | دسته‌بندی خودکار (پایین را ببینید) |
| `category_detail` | جزئیات دسته (مثلاً سرویس ناموجود، فیلدهای کم) |

**دسته‌بندی‌ها (`category`):** `unsupported_service` (سرویسی که نداریم — حالا فقط
ویلا/بوم‌گردی؛ تور و پرواز خارجی از این دسته درآمدند چون پشتیبانی می‌شوند) · `unsupported_city_for_service` (شهر را می‌شناسیم ولی سرویس ندارد) ·
`incomplete` (فیلد کم داشت) · `unknown_intent` (اصلاً نفهمید) · `help` (احوال‌پرسی/
راهنما) · خالی = این جمله به‌تنهایی کامل بود. **متن خام همیشه ذخیره می‌شود، حتی وقتی
دسته‌بندی شد.**

## ⚠️ جدول `parses` موفقیت را نمی‌سنجد — یک جمله را می‌سنجد

**این مهم‌ترین نکته‌ی تفسیری این سند است. اگر آن را ندانید، نرخ موفقیت را به‌شدت
کمتر از واقعیت برآورد می‌کنید.**

بک‌اند stateless است: هر جمله را **مستقل** پارس می‌کند و انباشت slotها سمت فرانت
انجام می‌شود. پس هر ردیف در `parses` فقط درباره‌ی همان یک جمله حرف می‌زند، نه درباره‌ی
کل مکالمه. این یعنی:

- **`url_from_this_utterance`** فقط می‌گوید همان یک جمله به‌تنهایی برای ساخت URL کافی
  بود یا نه. در هر مکالمه‌ی چندمرحله‌ای این ستون **۰** است، **حتی وقتی کاربر در نهایت
  موفق شد و جستجو را زد**.
- **`category`** هم همین‌طور: جمله‌های یک مکالمه‌ی موفق معمولاً `incomplete` ثبت می‌شوند.
  «خالی» یعنی *آن جمله* کامل بود، نه این‌که کاربر به نتیجه رسید.

**مثال واقعی (جریان دومرحله‌ای تست‌شده در ۲۰۲۶-۰۷-۲۸):** کاربر گفت «بلیط تهران به مشهد»،
دستیار تاریخ را پرسید، کاربر گفت «فردا»، کارت تأیید آمد و کاربر «جستجو کن» را زد —
یعنی **یک جریان کاملاً موفق**. چیزی که در `parses` ثبت شد:

| `raw_text` | `missing` | `url_from_this_utterance` | `category` |
|---|---|---|---|
| بلیط تهران به مشهد | `date` | ۰ | `incomplete` |
| بلیط فردا | `origin,destination` | ۰ | `incomplete` |

دو ردیف، هر دو ۰، هر دو `incomplete` — و صفر ردیفِ «حل‌شده». ولی همان نشست در جدول
`events` رویداد `search_clicked` را دارد.

**سیگنال درست موفقیت، رویداد `search_clicked` در جدول `events` است — نه هیچ ستونی در
`parses`.** جدول `parses` برای فهمیدن *چه چیزی* کاربران می‌گویند و کجا گیر می‌کنند
مفید است، نه برای شمردن موفقیت. برای هر سنجش نرخ، در سطح `session_id` کار کنید، نه
در سطح ردیف.

**`events`** — رویدادهای قیف که فرانت می‌فرستد (`/api/events`):
`clarification_shown` (detail=فیلدهای کم) · `clarification_answered` ·
`confirm_shown` · `search_clicked` (مهم‌ترین رویداد موفقیت) · `correction_clicked` ·
`new_search` · `error_shown` (detail=نوع) · `help_shown` · `voice_error`
(detail=کد خام Web Speech) · `cta_shown` · `cta_clicked` · `service_switched`.

**رویدادهای CTA (نجات از بن‌بست):** وقتی درخواستی را نمی‌توانیم انجام دهیم،
به‌جای پیام خشک، گزینه‌های جایگزین نشان داده می‌شود. این دو رویداد می‌گویند آیا
آن گزینه‌ها واقعاً کار می‌کنند:

- `cta_shown` — detail = `<نوع سناریو>:<جزئیات>`. نوع‌ها: `city_unsupported`
  (جزئیات = `<سرویس درخواستی>-><جایگزین‌های پیشنهادی>`) · `unsupported_service`
  (جزئیات = `villa`) · `unknown_place` (جزئیات = intent) · `unknown_intent`.
- `cta_clicked` — detail = `<نوع سناریو>:<گزینه‌ی انتخاب‌شده>`.

نسبت `cta_clicked` به `cta_shown` همان چیزی است که می‌گوید کاربر از بن‌بست نجات
پیدا کرد یا رها کرد.

**`service_switched`** — detail = `<سرویس قبلی>-><سرویس جدید>`. وقتی کاربر وسط یک
جریان، سرویس دیگری را صریحاً نام می‌برد ثبت می‌شود. نشان می‌دهد کاربران واقعاً بین
چه سرویس‌هایی جابه‌جا می‌شوند — مثلاً اگر «هتل → قطار» زیاد باشد یعنی مردم اول جای
اقامت را می‌بینند و بعد راه رسیدن.

## چطور کوئری بزنیم

```bash
ssh vta-server
sqlite3 /var/www/vta/shared/analytics.db     # حالت تعاملی؛ یا کوئری را مستقیم بده:
sqlite3 /var/www/vta/shared/analytics.db "SELECT ..."
```

### کوئری‌های آماده

```sql
-- ۱) پرتکرارترین متن‌هایی که اصلاً فهمیده نشدند
SELECT raw_text, COUNT(*) n FROM parses
WHERE category='unknown_intent'
GROUP BY raw_text ORDER BY n DESC LIMIT 30;

-- ۲) جفت‌های (سرویس، شهر) که پشتیبانی نشدند، پرتکرار اول
SELECT intent AS service, destination AS city, COUNT(*) n FROM parses
WHERE category='unsupported_city_for_service'
GROUP BY intent, destination ORDER BY n DESC;

-- ۳) سرویس‌های ناموجودی که کاربران خواستند (فعلاً فقط ویلا/بوم‌گردی)
SELECT category_detail AS service, COUNT(*) n FROM parses
WHERE category='unsupported_service'
GROUP BY category_detail ORDER BY n DESC;

-- ۴) قیف: چند نشست به کارت تأیید رسید و چند تا واقعاً «جستجو» زدند
SELECT
  (SELECT COUNT(DISTINCT session_id) FROM events WHERE type='confirm_shown')  AS reached_confirm,
  (SELECT COUNT(DISTINCT session_id) FROM events WHERE type='search_clicked') AS clicked_search;

-- ۴ب) نشست‌هایی که کارت تأیید دیدند ولی هرگز جستجو نزدند (رها کردن)
SELECT COUNT(*) AS abandoned_after_confirm FROM (
  SELECT session_id FROM events WHERE type='confirm_shown'
  EXCEPT
  SELECT session_id FROM events WHERE type='search_clicked'
);

-- ۵) پرتکرارترین فیلدهای کم (چه چیزی را کاربران معمولاً نمی‌گویند)
SELECT detail AS missing_fields, COUNT(*) n FROM events
WHERE type='clarification_shown'
GROUP BY detail ORDER BY n DESC;

-- ۶ب) نرخ نجات از بن‌بست: چند بار گزینه نشان دادیم و چند بار کلیک شد
SELECT
  (SELECT COUNT(*) FROM events WHERE type='cta_shown')   AS shown,
  (SELECT COUNT(*) FROM events WHERE type='cta_clicked') AS clicked;

-- ۶ج) کدام بن‌بست‌ها بیشتر پیش می‌آیند و کدام‌شان نجات پیدا می‌کنند
SELECT
  substr(detail, 1, instr(detail || ':', ':') - 1) AS scenario,
  SUM(type='cta_shown')   AS shown,
  SUM(type='cta_clicked') AS clicked
FROM events WHERE type IN ('cta_shown','cta_clicked')
GROUP BY scenario ORDER BY shown DESC;

-- ۶د) وقتی شهری سرویس درخواستی را ندارد، کاربران کدام جایگزین را می‌گیرند
SELECT detail, COUNT(*) n FROM events
WHERE type='cta_clicked' AND detail LIKE 'city_unsupported:%'
GROUP BY detail ORDER BY n DESC;

-- ۶ه) نشست‌هایی که بن‌بست دیدند و در نهایت جستجو کردند (نجات کامل)
SELECT COUNT(*) AS rescued FROM (
  SELECT session_id FROM events WHERE type='cta_clicked'
  INTERSECT
  SELECT session_id FROM events WHERE type='search_clicked'
);

-- ۶و) کاربران بین کدام سرویس‌ها جابه‌جا می‌شوند
SELECT detail AS switch, COUNT(*) n FROM events
WHERE type='service_switched'
GROUP BY detail ORDER BY n DESC;

-- ۶) توزیع خطاهای صوتی بر اساس کد خام (برای فهمیدن مشکل iOS با داده‌ی واقعی)
SELECT detail AS webspeech_code, COUNT(*) n FROM events
WHERE type='voice_error'
GROUP BY detail ORDER BY n DESC;

-- نمای کلی: تفکیک دسته‌ها. برچسب عمداً «تک‌جمله‌ی کامل» است نه «حل‌شده» —
-- این شمارش جمله‌هاست، نه مکالمه‌های موفق (بخش هشدار بالا را ببینید).
SELECT COALESCE(NULLIF(category,''),'(single-utterance complete)') AS category, COUNT(*) n
FROM parses GROUP BY category ORDER BY n DESC;

-- ۷) نرخ موفقیت واقعی — در سطح نشست، نه ردیف. این تنها راه درست است.
SELECT
  (SELECT COUNT(DISTINCT session_id) FROM parses)                             AS sessions_total,
  (SELECT COUNT(DISTINCT session_id) FROM events WHERE type='search_clicked') AS sessions_searched;

-- نمای کلی: شمارش رویدادها
SELECT type, COUNT(*) n FROM events GROUP BY type ORDER BY n DESC;
```

> نکته: `unknown_city` (شهری که خواسته ولی در واژگان ما نیست) عمداً از `unknown_intent`
> جدا نشده — تشخیص قابل‌اعتمادش به یک فهرست کامل شهرهای ایران نیاز دارد. فعلاً متن خامِ
> `unknown_intent` را بخوانید تا این موارد را دستی ببینید.
