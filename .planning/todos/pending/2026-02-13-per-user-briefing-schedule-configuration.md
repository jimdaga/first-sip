---
created: 2026-02-13T03:26:16.669Z
title: Per-user briefing schedule configuration
area: worker
files:
  - internal/config/config.go
  - internal/worker/scheduler.go
  - internal/worker/worker.go
  - internal/models/user.go
---

## Problem

Briefing schedule is currently system-wide — a single `BRIEFING_SCHEDULE` and `BRIEFING_TIMEZONE` environment variable controls when briefings generate for ALL users. This means every user gets their briefing at the same time regardless of their timezone or preference.

For a better UX, each user should be able to configure their own daily briefing time and timezone from their profile settings (e.g., "6 AM Pacific" vs "7 AM Eastern").

## Solution

1. Add `BriefingSchedule` (string, default "0 6 * * *") and `BriefingTimezone` (string, default "UTC") fields to the `User` model
2. Add a profile settings page where users can update their schedule preferences
3. Change the scheduler to run frequently (e.g. every minute) and check which users are "due" for a briefing based on their individual schedule/timezone, rather than firing one global cron
4. The existing `handleScheduledBriefingGeneration` loop already iterates over users — it needs a per-user time check filter added
5. Keep the global env vars as defaults for new users
