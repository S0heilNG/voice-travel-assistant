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
| `intent` | intent تشخیص‌داده‌شده (flight/train/bus/hotel/help/unknown) |
| `origin`, `destination` | نام فارسی شهرها (اگر استخراج شد) |
| `date`, `nights` | تاریخ شمسی و تعداد شب (هتل) |
| `missing` | فیلدهای کم (comma-separated) |
| `city_supported` | آیا شهر برای این سرویس پشتیبانی می‌شود |
| `search_url_built` | آیا در نهایت URL ساخته شد |
| `category` | دسته‌بندی خودکار (پایین را ببینید) |
| `category_detail` | جزئیات دسته (مثلاً سرویس ناموجود، فیلدهای کم) |

**دسته‌بندی‌ها (`category`):** `unsupported_service` (سرویسی که نداریم: تور/پرواز
خارجی/ویلا) · `unsupported_city_for_service` (شهر را می‌شناسیم ولی سرویس ندارد) ·
`incomplete` (فیلد کم داشت) · `unknown_intent` (اصلاً نفهمید) · `help` (احوال‌پرسی/
راهنما) · خالی = کاملاً حل شد. **متن خام همیشه ذخیره می‌شود، حتی وقتی دسته‌بندی شد.**

**`events`** — رویدادهای قیف که فرانت می‌فرستد (`/api/events`):
`clarification_shown` (detail=فیلدهای کم) · `clarification_answered` ·
`confirm_shown` · `search_clicked` (مهم‌ترین رویداد موفقیت) · `correction_clicked` ·
`new_search` · `error_shown` (detail=نوع) · `help_shown` · `voice_error`
(detail=کد خام Web Speech).

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

-- ۳) سرویس‌های ناموجودی که کاربران خواستند (تور، پرواز خارجی، ویلا)
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

-- ۶) توزیع خطاهای صوتی بر اساس کد خام (برای فهمیدن مشکل iOS با داده‌ی واقعی)
SELECT detail AS webspeech_code, COUNT(*) n FROM events
WHERE type='voice_error'
GROUP BY detail ORDER BY n DESC;

-- نمای کلی: تفکیک دسته‌ها
SELECT COALESCE(NULLIF(category,''),'(resolved)') AS category, COUNT(*) n
FROM parses GROUP BY category ORDER BY n DESC;

-- نمای کلی: شمارش رویدادها
SELECT type, COUNT(*) n FROM events GROUP BY type ORDER BY n DESC;
```

> نکته: `unknown_city` (شهری که خواسته ولی در واژگان ما نیست) عمداً از `unknown_intent`
> جدا نشده — تشخیص قابل‌اعتمادش به یک فهرست کامل شهرهای ایران نیاز دارد. فعلاً متن خامِ
> `unknown_intent` را بخوانید تا این موارد را دستی ببینید.
