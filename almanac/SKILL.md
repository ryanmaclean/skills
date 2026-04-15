---
name: almanac
description: >
  Almanac reference for dates, holidays, time zones, business days, and astronomical events.
  Use when checking if a date is a business day, calculating date differences, converting time zones,
  looking up holidays, or checking for astronomical events like solstices, equinoxes, and moon phases.
  Also for seasonal calculations, workday schedules, and calendar arithmetic.
---

# almanac — Date, Time, and Calendar Reference

Quick reference for calendar arithmetic, business days, holidays, and astronomical events.

## Business Days

**Business day**: Monday-Friday, excluding holidays.

**US Federal Holidays** (2026):
- New Year's Day: January 1
- Martin Luther King Jr. Day: Third Monday in January
- Presidents' Day: Third Monday in February
- Memorial Day: Last Monday in May
- Juneteenth: June 19
- Independence Day: July 4
- Labor Day: First Monday in September
- Columbus Day: Second Monday in October
- Veterans Day: November 11
- Thanksgiving Day: Fourth Thursday in November
- Christmas Day: December 25

**Observed holidays**: If a federal holiday falls on Saturday, it's observed on Friday. If on Sunday, observed on Monday.

## Time Zones

| Zone | UTC Offset | Major Cities |
|------|-----------|--------------|
| UTC | +0 | Greenwich |
| EST/EDT | -5/-4 | New York, Eastern US |
| CST/CDT | -6/-5 | Chicago, Central US |
| MST/MDT | -7/-6 | Denver, Mountain US |
| PST/PDT | -8/-7 | Los Angeles, Pacific US |
| JST | +9 | Tokyo |
| CET/CEST | +1/+2 | Berlin, Central Europe |
| BST | +1 | London (summer) |
| GMT | +0 | London (winter) |

## Astronomical Events (2026)

| Event | Date |
|-------|------|
| Spring Equinox | March 20 |
| Summer Solstice | June 21 |
| Autumn Equinox | September 22 |
| Winter Solstice | December 21 |

**Moon phases** (approximate):
- New moon: Every 29.5 days
- First quarter: ~7 days after new
- Full moon: ~14-15 days after new
- Last quarter: ~22 days after new

## Date Calculations

**Workday calculations**:
- Add/subtract business days (skip weekends and holidays)
- Check if a date is a business day
- Find next/previous business day

**Date arithmetic**:
- Days between dates
- Add/subtract days, weeks, months, years
- Week of year (ISO 8601: week starts Monday, week 1 contains first Thursday)

## Seasonal Information

| Season (Northern Hemisphere) | Months |
|------------------------------|--------|
| Spring | March, April, May |
| Summer | June, July, August |
| Autumn | September, October, November |
| Winter | December, January, February |

## Scripts

| Script | Use |
|--------|-----|
| `script/is-business-day.nu "2026-04-15"` | Check if date is a business day |
| `script/add-business-days.nu "2026-04-15" 5` | Add 5 business days to date |
| `script/days-between.nu "2026-04-01" "2026-04-15"` | Calculate days between dates |
| `script/week-of-year.nu "2026-04-15"` | Get ISO week number for date |
| `script/moon-phase.nu "2026-04-15"` | Get moon phase for date |

Scripts return structured JSON or human-readable output based on flags.

## Common Patterns

**Check if today is a business day**:
```bash
nu script/is-business-day.nu (date now | format date "%Y-%m-%d")
```

**Calculate project deadline** (10 business days from today):
```bash
nu script/add-business-days.nu (date now | format date "%Y-%m-%d") 10
```

**Check if a deadline falls on a weekend**:
```bash
nu script/is-business-day.nu "2026-04-12"  # Saturday
```

## Time Zone Conversion

When converting time zones:
1. Always specify source and target zones
2. Account for DST transitions
3. Use UTC as the canonical intermediate format

Example: Convert 2026-04-15 14:00 EST to JST
- EST → UTC: +5 hours → 2026-04-15 19:00 UTC
- UTC → JST: +9 hours → 2026-04-16 04:00 JST

## Leap Years

A year is a leap year if:
- Divisible by 4, but not by 100, OR
- Divisible by 400

2024: leap year
2025: not
2026: not
2028: leap year
