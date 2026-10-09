# 🕌 Namaz Time Bot — User Guide

Welcome to the **Namaz Time Bot** user guide! This manual explains how to use the bot in Telegram, check prayer schedules, set up daily reminders, customize notifications, and use the bot in Telegram groups and channels.

---

## 📑 Table of Contents

1. [Quick Start](#-quick-start)
2. [Checking Prayer Times](#-checking-prayer-times)
3. [Choosing Your City or District](#-choosing-your-city-or-district)
4. [Setting Up Daily Notifications & Reminders](#-setting-up-daily-notifications--reminders)
5. [Multi-City Schedule Digest](#-multi-city-schedule-digest)
6. [Hadith of the Day & Islamic Reminders](#-hadith-of-the-day--islamic-reminders)
7. [Using the Bot in Telegram Groups](#-using-the-bot-in-telegram-groups)
8. [Publishing Prayer Schedules in Telegram Channels](#-publishing-prayer-schedules-in-telegram-channels)
9. [Commands Summary](#-commands-summary)
10. [Frequently Asked Questions (FAQ)](#-frequently-asked-questions-faq)

---

## 🚀 Quick Start

1. Open Telegram and search for the bot (or click the invite link provided to you).
2. Press **Start** (or send `/start`).
3. The bot greets you with the main menu and your currently selected city (default: **Ufa**).
4. Tap **"🕌 Today's Schedule"** to immediately view today's prayer times.

---

## 🕌 Checking Prayer Times

You can view today's schedule at any time:
- Send the command `/today`, or
- Tap the **"🕌 Today's Schedule"** button on the bottom keyboard.

### How to Read the Schedule Card:

Each card shows:
- **Location & Dates**: The city name, Gregorian date, day of the week, and the current **Hijri (lunar) date**.
- **Fajr (Фаджр)**: Morning prayer and the final moment for **Suhoor** (pre-dawn meal before fasting).
- **Sunrise (Восход / Шурук)**: Exact moment of sunrise. Morning prayer time ends here. *Voluntary prayers are prohibited during sunrise.*
- **Dhuhr (Зухр)**: Noon prayer.
- **Asr (Аср)**: Afternoon prayer.
- **Maghrib (Магриб)**: Sunset prayer and the time for **Iftar** (breaking the fast).
- **Isha (Иша)**: Night prayer.

---

## 🏙 Choosing Your City or District

The bot supports **21 cities** and **40 municipal districts** across the Republic of Bashkortostan.

1. Send `/city` or tap **"🏙 Select City"**.
2. An interactive window with buttons will appear:
   - Tap **"🏙 Cities"** to browse republican cities (e.g. *Ufa, Sterlitamak, Salavat, Neftekamsk, Oktyabrsky, Sibay*, etc.).
   - Tap **"🗺 Districts"** to browse municipal districts (e.g. *Abzelilovsky, Beloretsky, Chishminsky, Iglinsky*, etc.).
3. Use the pagination arrows (`«` and `»`) to navigate through pages.
4. Click on your city or district.
5. The bot will instantly confirm your choice. All future schedules and alerts will automatically use this location.

---

## ⏰ Setting Up Daily Notifications & Reminders

You can configure exactly when and how the bot notifies you.

Open the settings menu by sending `/settings` or tapping **"⚙️ Settings"**.

### Available Notification Options:

1. **Daily Timetable Broadcast (Daily Digest)**:
   - Automatically sends you the complete schedule for the day.
   - You can choose default preset times (e.g. `06:00`, `20:00`) or send a custom time in `HH:MM` format.
   - You can configure multiple broadcast times per day.
   - Use `/subscribe` to quickly turn on daily broadcasts or `/unsubscribe` to pause them.

2. **15-Minute Advance Reminder (⏰)**:
   - Sends a friendly reminder **15 minutes before** the start of each prayer so you have time to perform ablution (wudu) and prepare.
   - Can be turned on or off with a single tap in the `/settings` menu.

3. **Exact Time Notification (🔔)**:
   - Sends an alert at the exact minute each prayer begins.
   - Can be toggled on/off in `/settings`.

4. **Hadith Attachments**:
   - Optionally attaches an authentic hadith about the virtue of that specific prayer to your notification.

5. **Sunnah Fasting Reminders (🌕)**:
   - Reminders with hadiths on Mondays, Thursdays, and the "White Days" (13th, 14th, 15th of Hijri month).
   - Can be turned on or off independently via `/settings`.

---

## 📋 Multi-City Schedule Digest

If you frequently travel or have family members living in different towns across Bashkortostan, you can use the **Digest** feature.

1. Send `/digest` or tap **"📋 Schedule Digest"**.
2. The bot displays compact prayer cards for all your favorite cities at once.
3. Managing your cities:
   - Tap **"➕ Add city to digest"** to pick an additional city from the list.
   - Tap **"✕ [City Name]"** to remove a city from your digest.
4. **Sharing**:
   - Below each card, there is a **"↗️ Share"** button that lets you forward formatted timetables directly to WhatsApp, Telegram chats, or family groups with a single click.

---

## 📖 Hadith of the Day & Islamic Reminders

### Daily Hadith on Demand:
- Send `/hadith` or tap **"📖 Hadith of the Day"**.
- The bot sends a verified hadith with authentic references (*Sahih al-Bukhari, Sahih Muslim, Riyad as-Salihin*).

### Automated Islamic Occasion Reminders:
The bot automatically sends inspirational hadiths and reminders for:
- **Sunnah Fasting Days**:
  - Reminders on the eve of Mondays and Thursdays.
  - Reminders for the **"White Days"** (*Ayyam al-Beed* — 13th, 14th, and 15th of every Hijri month).
- **Blessed Lunar Months**:
  - Welcoming reminders and fasting advice ahead of sacred months: *Ramadan, Shawwal, Dhul-Hijjah, Muharram, Rajab, Sha'ban, and Dhul-Qi'dah*.

---

## 👥 Using the Bot in Telegram Groups

Bring automated prayer timetables to your community, mosque, or family group chat!

### Step 1: Add the Bot
1. Open your Telegram group chat.
2. Tap the group profile → **Add Members**.
3. Search for the bot and add it.

### Step 2: Grant Administrator Rights
1. Promote the bot to **Administrator**.
2. Ensure the bot has permissions to:
   - **Send Messages**
   - **Delete Messages** (used to automatically clean up bot commands and keep the chat tidy).

### Step 3: Configure Settings Privately
1. Inside the group, send `/settings` or `/city`.
2. The bot will automatically delete the command and post an ephemeral message with a button: **«⚙️ Configure Bot in PM»**.
3. Click the button to open a private chat with the bot.
4. In private messages, choose the city for the group, set the broadcast schedule, and configure reminders.
5. Only group administrators have permission to modify group settings.

---

## 📢 Publishing Prayer Schedules in Telegram Channels

Automate daily prayer timetable posts for your mosque or Islamic Telegram channel.

### Step 1: Add Bot to Channel
1. Go to your Telegram channel → **Channel Settings** → **Administrators**.
2. Add the bot as an Administrator with the **"Post Messages"** permission.

### Step 2: Connect the Channel
Choose either method:
- **Method A**: Forward any post from your channel to the bot in a private message.
- **Method B**: Send the command `/channel @your_channel_name` to the bot in a private chat.

### Step 3: Customize Channel Posts
In the bot's private chat, open channel settings to customize:
- **City / District**: Set the location for the schedule.
- **Broadcast Time**: Choose when the bot posts the daily schedule to your channel.
- **Custom Header**: Add custom text at the top of the post (e.g. *«🕌 Central Mosque Schedule»*).
- **Custom Footer**: Add your channel links, du'a, or announcements at the bottom.
- **Prayer Names Style**:
  - Standard Russian (*Фаджр, Восход, Зухр, Аср, Магриб, Иша*)
  - Bashkir / Tatar (*Иртәнге, Кояш, Өйлә, Икенде, Ашам, Ястү*)
  - Traditional Arabic transcription
  - Fully custom names

---

## 📋 Commands Summary

| Command | Where It Works | Description |
|---|---|---|
| `/start` | Private Chat | Start the bot and display the main interactive menu |
| `/today` | Private / Group | Show today's prayer timetable |
| `/digest` | Private Chat | Show multi-city schedule cards with quick share buttons |
| `/city` | Private / Group | Select your city or municipal district |
| `/settings` | Private / Group | Manage reminder times, alerts, and formatting |
| `/hadith` | Private / Group | Read an authentic Hadith of the Day |
| `/channels` | Private Chat | View and manage all your connected groups and channels |
| `/channel` | Private Chat | Connect a Telegram channel (`/channel @username`) |
| `/subscribe` | Private / Group | Turn on automatic daily timetable delivery |
| `/unsubscribe` | Private / Group | Turn off daily timetable delivery |

---

## ❓ Frequently Asked Questions (FAQ)

### Where does the bot get prayer times?
Prayer times are obtained directly from the official **DUM RB API** (Spiritual Administration of Muslims of the Republic of Bashkortostan). Sunrise times are calculated using astronomical coordinate models for each specific locality.

### Why do some prayer times differ from mobile apps?
Generic mobile apps often use worldwide mathematical estimation formulas (such as MWL, ISNA, or Egyptian General Authority). This bot uses the **official regional standard established by DUM RB**, taking into account local high-latitude twilight nuances specific to Bashkortostan.

### How do I stop receiving messages?
Send the command `/unsubscribe` or go to `/settings` and disable your notification toggles. You can also turn off notifications at any time without deleting the bot.

### How do I change the time when the daily schedule is sent?
Go to `/settings` → **Daily Schedule Time** and pick one of the preset times or type your own custom time (e.g. `06:30` or `21:00`).

### Can I use the bot if I live outside Ufa?
Yes! The bot supports all **21 cities** and **40 districts** in Bashkortostan. Use `/city` to pick your specific town or district.
