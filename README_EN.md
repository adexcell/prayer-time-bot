# 🕌 Namaz Time Bot (Republic of Bashkortostan)

A high-performance Telegram bot written in **Go (Golang)** providing accurate, official Islamic prayer schedules for all cities and districts of the Republic of Bashkortostan. Data is sourced directly from the official **DUM RB API** (Spiritual Administration of Muslims of the Republic of Bashkortostan) with astronomical sunrise calculations.

> 📖 **Looking for the bot user manual? Check the [Telegram Bot User Guide](USER_GUIDE_EN.md)**.  
> 🇷🇺 **Русская версия документации: [README.md](README.md)**.

---

## ✨ Features & Capabilities

- **🕌 Accurate Daily Prayer Schedule**:
  - Displays all obligatory prayers and key daily milestones: *Fajr (End of Suhoor)*, *Sunrise (Shuruq)*, *Dhuhr*, *Asr*, *Maghrib (Iftar)*, and *Isha*.
- **🏙 Full Regional Coverage**:
  - Supports **21 cities** and **40 districts** across the Republic of Bashkortostan.
  - Interactive inline keyboard interface with tabs ("Cities" / "Districts") and paginated navigation.
  - Location selection is stored persistently per user/group/channel.
- **⏰ Flexible Reminders & Automated Broadcasts**:
  - **Daily Scheduled Summaries**: Send the full daily timetable at configured times (e.g. `06:00`, `20:00`, or arbitrary custom hours/minutes).
  - **15-Minute Advance Reminder**: Helpful notification 15 minutes before the start of each prayer.
  - **Exact Time Notification**: Alert sent precisely at the minute each prayer time arrives.
- **📋 Multi-City Digest (`/digest`)**:
  - Generate formatted schedule cards for multiple favorite cities/districts simultaneously.
  - One-click share buttons formatted for easy forwarding to WhatsApp, Telegram channels, and other messengers.
- **📖 Authentic Hadiths from Canonical Collections**:
  - Collections include: **Riyad as-Salihin** (Imam an-Nawawi), **Sahih al-Bukhari**, and **Sahih Muslim**.
  - **Fasting Reminders**: Automated reminders for Sunnah fasts on the White Days (13th, 14th, and 15th of the Hijri month) as well as Mondays and Thursdays.
  - **Sacred Month Notifications**: Welcoming reminders before blessed lunar months (Ramadan, Shawwal, Dhul-Hijjah, Muharram, Rajab, Sha'ban, Dhul-Qi'dah).
  - **Prayer-Attached Hadiths**: Relevant hadiths about specific prayers (Fajr, Dhuhr, Asr, Maghrib, Isha, Jumu'ah) attached to prayer time notifications.
  - **Hadith of the Day**: Daily automated broadcast and on-demand retrieval via `/hadith`.
- **📢 Telegram Groups & Channels Support**:
  - Full support for automated daily broadcasts and reminders in public/private channels and discussion groups.
  - Group anti-spam redirect: configuring group settings redirects group admins to private messages with the bot.
  - Customizable message styles: custom headers, footers, prayer naming presets, and custom names.
- **🛠 Admin Management Panel (`/admin`)**:
  - Interactive GUI for bot administrators.
  - Prayer time manual offsets (+/- minutes or fixed time per city and prayer).
  - Hijri calendar offset adjustments.
  - Hadith catalog browser and management (add, edit, toggle collections and categories).
  - Welcome message template customization with `{city}` placeholder.
  - Audience statistics and system status.
- **💾 CGO-Free SQLite Storage**:
  - Built on `modernc.org/sqlite` — requires no C compiler (gcc) or external database servers.

---

## 🛠 Tech Stack

- **Language**: [Go (Golang)](https://go.dev/) (v1.22+)
- **Telegram Bot API**: [`github.com/go-telegram-bot-api/telegram-bot-api/v5`](https://github.com/go-telegram-bot-api/telegram-bot-api)
- **Database**: SQLite via [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) (pure Go, CGO-free)
- **Task Scheduler**: [`github.com/robfig/cron/v3`](https://github.com/robfig/cron)
- **Configuration**: [`github.com/kelseyhightower/envconfig`](https://github.com/kelseyhightower/envconfig), [`github.com/joho/godotenv`](https://github.com/joho/godotenv)
- **Data Providers**:
  - Official DUM RB API (`https://api.dumrb.com`)
  - Astronomical Sunrise (`https://voshod-solnca.ru`)

---

## 📁 Repository Structure

```text
namaz-time-bot/
├── config/
│   └── config.go            # Environment variables configuration loader and validator
├── internal/
│   ├── api/
│   │   └── dumrb.go         # DUM RB API client, city/district listings, message formatting
│   ├── bot/
│   │   ├── admin.go         # Admin control panel, prayer offsets, calendar adjustments
│   │   ├── admin_hadiths.go # Hadith catalog manager and delivery settings
│   │   ├── bot.go           # Core bot controller, routing, and user interactions
│   │   ├── channel.go       # Channel setup and message publishing handlers
│   │   └── group.go         # Group setup, templates, custom styling, anti-spam redirect
│   ├── scheduler/
│   │   └── cron.go          # Background cron jobs (reminders, broadcasts, hadiths in UTC+5)
│   └── storage/
│       └── sqlite.go        # SQLite database initialization, migrations, CRUD operations
├── .env.example             # Example environment variables file
├── .gitignore               # Ignored files (binaries, databases, credentials)
├── Makefile                 # Build, run, test, and maintenance automation
├── go.mod                   # Go module definition
├── go.sum                   # Go dependencies checksums
├── main.go                  # Application entry point
├── README.md                # Russian documentation
└── README_EN.md             # English documentation (this file)
```

---

## 🤖 Bot Commands Reference

| Command | Scope | Description |
|---|---|---|
| `/start` | Private | Welcome message and interactive main menu |
| `/today` | Private / Group | Today's prayer timetable for the selected location |
| `/digest` | Private | Multi-city schedule cards with quick share buttons |
| `/city` | Private / Group | Select city or municipal district of Bashkortostan |
| `/settings` | Private / Group | Configure reminder times, notifications, and styles |
| `/hadith` | Private / Group | Get an authentic Hadith of the Day |
| `/channels` | Private | View and manage your connected channels and groups |
| `/channel` | Private | Connect a Telegram channel (`/channel @channel_username`) |
| `/subscribe` | Private / Group | Enable daily automated timetable broadcast |
| `/unsubscribe` | Private / Group | Disable daily automated timetable broadcast |
| `/admin` | Private | Super-admin control panel (adjustments, hadiths, stats) |
| `/addadmin <id>` | Private | Grant administrator rights to a Telegram user ID |
| `/deladmin <id>` | Private | Revoke administrator rights from a Telegram user ID |
| `/hijri <offset>`| Private | Adjust the global Hijri calendar day offset (+/- days) |

---

## 📖 User Guide

### 1. Selecting Location (`/city`)
Send `/city` to open an inline browser:
- Switch between **«🏙 Cities»** (21 cities) and **«🗺 Districts»** (40 municipal districts).
- Use pagination buttons (`«`, `»`) to navigate through pages.
- Click any location name to immediately set it as your default.

### 2. Multi-City Digest (`/digest`)
- Shows prayer cards for all cities saved in your favorites list.
- Use **«➕ Add city to digest»** or delete entries using **«✕ [City]»**.
- Each card includes a share button to quickly forward formatted text to WhatsApp groups or other chats.

### 3. Setting Up Notifications (`/settings`)
In `/settings`, you can toggle:
- **Daily Broadcast**: Set one or multiple delivery times (e.g., `06:00`, `21:30`).
- **15-Minute Advance Alert**: Enable/disable audio/text notification 15 minutes before prayer time.
- **Exact Time Notification**: Enable/disable notification at the exact moment of each prayer.
- **Hadith Notifications**: Enable daily hadith broadcasts and hadith attachments to prayer alerts.

---

## 👥 Telegram Groups & Channels Guide

### Setting Up a Group
1. Add the bot to your Telegram group.
2. Grant the bot **Administrator permissions** (specifically *Send Messages* and *Delete Messages*).
3. Type `/settings` or `/city` inside the group.
4. To prevent cluttering the chat, the bot will delete your command and reply with a self-destructing button: **«⚙️ Configure in Private Messages»**.
5. Click the button to configure the group's city, broadcast times, and notification preferences privately.
6. Only group administrators can alter group settings or toggle `/subscribe` / `/unsubscribe`.

### Setting Up a Telegram Channel
1. Add the bot to your Telegram channel as an **Administrator** with **Post Messages** rights.
2. Either:
   - Forward any post from your channel to the bot in a private message.
   - Or send `/channel @your_channel_username` directly to the bot in private.
3. The bot will detect your channel and open the customization dashboard.
4. Available channel settings:
   - Location (City or District).
   - Scheduled broadcast times (send once or multiple times daily).
   - Custom Header (text shown above the prayer timetable).
   - Custom Footer (text/links shown beneath the schedule).
   - Prayer Name Style (Standard Russian, Bashkir/Tatar, Traditional, or fully custom names).

---

## 👑 Administrator Guide (`/admin`)

Access to the administrator panel is restricted to users listed in `ADMIN_IDS` in `.env`, or added via `/addadmin`.

### Key Admin Tools:
1. **Prayer Time Offsets**:
   - Apply +/- minute corrections to any prayer in any city (e.g., +2 minutes to Fajr in Ufa).
   - Set fixed prayer times if needed.
2. **Hijri Calendar Calibration (`/hijri`)**:
   - Due to moon-sighting variations, lunar dates may differ by ±1 or ±2 days.
   - Admins can shift the date globally using `/hijri <offset>` (e.g. `/hijri +1` or `/hijri -1`) or via the interactive UI.
3. **Hadith Management**:
   - Browse catalog across categories (Prayer, Fasting, Ramadan, General Adab).
   - Add new verified hadiths specifying the Arabic/Russian text, collection, and reference number.
   - Adjust daily hadith delivery times.
4. **Welcome Greeting**:
   - Customize the template sent to new users on `/start` using the `{city}` placeholder.
5. **Audience Statistics**:
   - View real-time active subscriber counts, group counts, and channel counts.

---

## 🚀 Installation & Local Setup

### 1. Prerequisites
- **Go 1.22+** ([download from golang.org](https://go.dev/dl/))
- Telegram bot token from [@BotFather](https://t.me/BotFather)

### 2. Clone the Repository
```bash
git clone https://github.com/your-username/namaz-time-bot.git
cd namaz-time-bot
```

### 3. Configure Environment Variables
Copy the template file:
```bash
cp .env.example .env
```

Edit `.env` and fill in your values:
```env
# Required Telegram bot token from @BotFather
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGhIJKlmNoPQRsTUVwxyZ

# Default fallback city
CITY=Уфа

# Country and calculation method
COUNTRY=Russia
METHOD=14

# Comma-separated Telegram User IDs for super-administrators
ADMIN_IDS=123456789,987654321
```

### 4. Install Dependencies
```bash
go mod download
```

### 5. Running the Bot

#### Development Mode:
```bash
go run main.go
# or using Makefile:
make run
```

#### Build Optimized Executable:
```bash
make build
# or manual Go build:
go build -ldflags="-w -s" -o bin/namaz-bot main.go

# Run the binary:
./bin/namaz-bot
```

### 6. Helpful `make` Commands
| Command | Action |
|---|---|
| `make build` | Compiles stripped production binary to `bin/namaz-bot` |
| `make run` | Runs bot directly with `go run` |
| `make test` | Runs unit tests with race detection (`-race`) |
| `make test-coverage` | Runs tests and prints coverage metrics |
| `make fmt` | Formats code with `gofmt` |
| `make tidy` | Syncs and prunes `go.mod` and `go.sum` |
| `make check` | Runs formatting, tidy, and tests |
| `make clean` | Removes built binaries and temporary test artifacts |

---

## 🌐 Production Server Deployment (Systemd)

To deploy the bot as a persistent daemon on a Linux server:

1. **Compile the binary for Linux**:
   ```bash
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/namaz-bot main.go
   ```

2. **Prepare directories on the server**:
   ```bash
   sudo useradd -r -s /bin/false namazbot || true
   sudo mkdir -p /opt/namaz-bot
   sudo cp bin/namaz-bot /opt/namaz-bot/
   sudo cp .env /opt/namaz-bot/
   sudo chown -R namazbot:namazbot /opt/namaz-bot
   ```

3. **Create the systemd service file** at `/etc/systemd/system/namaz-bot.service`:
   ```ini
   [Unit]
   Description=Namaz Time Telegram Bot
   After=network.target network-online.target
   Wants=network-online.target

   [Service]
   Type=simple
   User=namazbot
   Group=namazbot
   WorkingDirectory=/opt/namaz-bot
   ExecStart=/opt/namaz-bot/namaz-bot
   Restart=always
   RestartSec=5s

   # Security hardening
   ProtectSystem=full
   ProtectHome=true
   NoNewPrivileges=true

   [Install]
   WantedBy=multi-user.target
   ```

4. **Enable and start the service**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now namaz-bot
   ```

5. **Monitor logs in real time**:
   ```bash
   journalctl -u namaz-bot -f -o cat
   ```

---

## 📄 License

This project is licensed under the **MIT License**.
