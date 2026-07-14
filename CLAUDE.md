# voice-travel-assistant

دستیار سفر صوتی فارسی‌زبان که به جستجوی پرواز و هتل پلتفرم ۷۸۰ وصل می‌شود.

## معماری

پروژه از سه بخش مستقل تشکیل شده:

- **backend/** — سرویس Go (Fiber). مسئول API، ماژول NLU برای تشخیص intent فارسی، و هماهنگی بین فرانت‌اند و automation-service.
- **automation-service/** — میکروسرویس Node.js با Playwright. مسئول اتوماسیون واقعی (جستجوی پرواز/هتل روی پلتفرم ۷۸۰). بک‌اند Go با این سرویس از طریق HTTP داخلی ارتباط برقرار می‌کند.
- **frontend/** — اپلیکیشن React با پشتیبانی کامل RTL فارسی و فونت Vazirmatn. ورودی صدا از طریق Web Speech API مستقیماً سمت کلاینت انجام می‌شود (بدون نیاز به سرویس STT جداگانه).

## چرا این تصمیمات فنی

- **Playwright در سرویس جدا (نه در Go):** Playwright کتابخانه رسمی Go ندارد. به همین دلیل تمام منطق اتوماسیون مرورگر در یک میکروسرویس Node.js مجزا (`automation-service`) پیاده‌سازی شده و بک‌اند Go فقط از طریق HTTP آن را صدا می‌زند. این جداسازی باعث می‌شود بک‌اند اصلی سبک و مستقل از وابستگی‌های سنگین Playwright/Chromium بماند.
- **Web Speech API سمت کلاینت:** تشخیص گفتار در مرورگر کاربر انجام می‌شود، نه در بک‌اند — نیازی به ارسال استریم صوتی به سرور نیست.
- **ماژول NLU در بک‌اند Go:** تشخیص intent فارسی (مثلاً "بلیط تهران به مشهد برای فردا") در `backend/internal/nlu` انجام می‌شود و نتیجه به‌صورت پارامترهای ساخت‌یافته به `automation-service` پاس داده می‌شود.

## ساختار پوشه‌ها

```
voice-travel-assistant/
├── backend/                  # Go (Fiber)
│   ├── cmd/server/main.go    # entry point
│   ├── internal/
│   │   ├── nlu/               # تشخیص intent فارسی
│   │   ├── automation/        # کلاینت HTTP به automation-service
│   │   ├── handlers/          # HTTP handlers (health, ...)
│   │   └── config/            # بارگذاری env/config
│   ├── .env.example
│   └── go.mod
├── automation-service/       # Node.js + Playwright
│   ├── src/
│   └── package.json
├── frontend/                  # React (RTL + Vazirmatn + Web Speech API)
│   ├── src/
│   └── package.json
├── deploy.sh                  # اسکریپت دیپلوی خودکار (نگاه کنید به بخش «دیپلوی»)
└── CLAUDE.md
```

## دیپلوی

سرور تولید: `vta-server` (VPS به IP `194.62.43.142`، Ubuntu 26.04)، با کاربر اپلیکیشن غیر-root به نام `vtaapp`.

**از این به بعد دیپلوی همیشه از طریق اسکریپت خودکار انجام می‌شود، نه به‌صورت دستی:**

```
ssh vta-server "su - vtaapp -c '/var/www/vta/deploy.sh main'"
```

(آرگومان اول نام برنچ است؛ در نبود آرگومان پیش‌فرض `main` است.)

`deploy.sh` (که نسخه‌ی مرجعش همین‌جا در ریشه‌ی ریپو نگه‌داری می‌شود و با `cat` مستقیم روی سرور در مسیر `/var/www/vta/deploy.sh` هم قرار دارد) این مراحل را خودکار انجام می‌دهد:

1. یک release جدید با نام `v-<تاریخ>-<ساعت>` در `/var/www/vta/releases/` می‌سازد و ریپو را (با retry در برابر قطعی‌های شبکه‌ای گیت‌هاب) در آن clone می‌کند.
2. بک‌اند Go را build می‌کند و `.env` واقعی را از `/var/www/vta/shared/.env` (که بین release‌ها مشترک و خارج از گیت است) در release کپی می‌کند.
3. اگر `automation-service` دیپندنسی واقعی داشته باشد، `npm install --production` می‌زند.
4. symlink مسیر `/var/www/vta/current` را به‌صورت atomic به release جدید تغییر می‌دهد و سرویس systemd (`vta-backend`) را ری‌استارت می‌کند.
5. health check می‌زند (`GET /health`)؛ در صورت شکست، به‌طور خودکار symlink و سرویس را به release قبلی برمی‌گرداند (rollback) و با کد خروج غیرصفر خارج می‌شود.
6. در صورت موفقیت، فقط ۳ release آخر را نگه می‌دارد و بقیه را پاک می‌کند.

جزئیات زیرساخت سرور (Go، Node، Nginx reverse-proxy روی پورت ۸۰، Docker، UFW، systemd unit در `/etc/systemd/system/vta-backend.service`) در این فایل مستند نیست و باید در کانفیگ خود سرور بررسی شود.

## وضعیت فعلی

فقط اسکلت اولیه (scaffold) پیاده‌سازی شده:
- `GET /health` در بک‌اند Go که `{"status": "ok"}` برمی‌گرداند.
- `go.mod`, `package.json` های frontend و automation-service آماده‌اند ولی بدون دیپندنسی نصب‌شده.

فیچرهای اصلی (NLU، اتصال به پلتفرم ۷۸۰، UI صوتی) هنوز پیاده‌سازی نشده‌اند و مرحله به مرحله اضافه می‌شوند.
