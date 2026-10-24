---
description: "Manage Datadog On-Call scheduling and rotations"
argument-hint: "--action <action> [--schedule-id ID] [options]"
---

# Datadog On-Call Scheduling Management

Manage on-call schedules and rotations for team coverage. Create schedules, configure rotations, and track who's currently on-call.

## What is On-Call Scheduling?

Datadog On-Call provides:
- **Schedule Management** - Create and manage on-call rotations
- **Team Coverage** - Ensure 24/7 team availability
- **Rotation Configuration** - Daily, weekly, biweekly, or monthly rotations
- **Timezone Support** - Global team coordination
- **Incident Response** - Automatic routing to on-call personnel

**Official Documentation**: https://docs.datadoghq.com/service_management/on-call/schedules/

## Usage

### Schedule Management

```bash
# List all on-call schedules
dd on-call --action list

# Create new schedule with weekly rotation
dd on-call --action create \
  --name "Backend Team Schedule" \
  --team-id team-123 \
  --rotation weekly \
  --members "user1,user2,user3" \
  --timezone "America/New_York"

# Create daily rotation
dd on-call --action create \
  --name "Daily Rotation" \
  --rotation daily \
  --members "user1,user2"

# Get schedule details
dd on-call --action get --schedule-id schedule-123

# Update schedule
dd on-call --action update \
  --schedule-id schedule-123 \
  --name "Updated Schedule Name" \
  --timezone "Europe/London"

# Delete schedule
dd on-call --action delete --schedule-id schedule-123
```

### Check Who's On-Call

```bash
# Check all schedules
dd on-call --action who

# Check specific schedule
dd on-call --action who --schedule-id schedule-123

# Get JSON output
dd on-call --action list --json
```

## Actions

- **list**, **list-schedules** - List all on-call schedules
- **create**, **create-schedule** - Create new on-call schedule
- **get**, **get-schedule** - Get schedule details
- **update**, **update-schedule** - Update schedule
- **delete**, **delete-schedule** - Delete schedule
- **who**, **who-is-on-call** - Show who is currently on-call

## Rotation Types

**daily** - 24-hour rotations
**weekly** - 7-day rotations (default)
**biweekly** - 14-day rotations
**monthly** - 30-day rotations

## Common Timezones

- **UTC** - Coordinated Universal Time
- **America/New_York** - Eastern Time (US)
- **America/Los_Angeles** - Pacific Time (US)
- **America/Chicago** - Central Time (US)
- **Europe/London** - UK Time
- **Europe/Paris** - Central European Time
- **Asia/Tokyo** - Japan Time
- **Asia/Singapore** - Singapore Time
- **Australia/Sydney** - Australian Eastern Time

## Use Cases

### 1. Create Weekly Rotation Schedule

```bash
dd on-call --action create \
  --name "Platform Team Weekly" \
  --team-id team-456 \
  --rotation weekly \
  --members "alice,bob,charlie" \
  --timezone "America/New_York" \
  --description "Weekly rotation for platform support"
```

Create rotating on-call schedule with automatic handoffs.

### 2. Set Up Daily Rotation

```bash
dd on-call --action create \
  --name "Daily NOC Coverage" \
  --rotation daily \
  --members "user1,user2,user3,user4,user5,user6,user7" \
  --timezone "UTC"
```

Configure 24-hour rotation for operations center.

### 3. Global Team Coverage

```bash
# APAC schedule
dd on-call --action create \
  --name "APAC Support" \
  --rotation weekly \
  --members "apac-team-1,apac-team-2" \
  --timezone "Asia/Tokyo"

# EMEA schedule
dd on-call --action create \
  --name "EMEA Support" \
  --rotation weekly \
  --members "emea-team-1,emea-team-2" \
  --timezone "Europe/London"

# Americas schedule
dd on-call --action create \
  --name "Americas Support" \
  --rotation weekly \
  --members "amer-team-1,amer-team-2" \
  --timezone "America/New_York"
```

Configure follow-the-sun support across regions.

### 4. Check Current On-Call Status

```bash
# Quick check of all schedules
dd on-call --action who

# Detailed schedule information
dd on-call --action get --schedule-id schedule-123 --json
```

Identify who is currently responsible for incidents.

### 5. Update Rotation Members

```bash
# Update schedule with new members
dd on-call --action update \
  --schedule-id schedule-123 \
  --members "user1,user2,user3,user4"
```

Adjust rotation as team membership changes.

### 6. Seasonal Schedule Adjustments

```bash
# Update timezone for daylight saving
dd on-call --action update \
  --schedule-id schedule-123 \
  --timezone "America/New_York"

# Update rotation frequency
dd on-call --action update \
  --schedule-id schedule-123 \
  --rotation biweekly
```

Adapt schedules to changing needs.

## Why Use the CLI?

- **Fast schedule creation** - Set up rotations in seconds
- **Automation** - Script schedule management
- **CI/CD integration** - Configure on-call as code
- **Team coordination** - Manage global coverage
- **Rotation flexibility** - Daily, weekly, biweekly, monthly options
- **JSON output** - Parse for further automation

## Example Prompts

> "Create an on-call schedule for my team"
> "Set up weekly rotation with 3 engineers"
> "Who is currently on-call?"
> "Update schedule timezone to Pacific time"
> "List all on-call schedules"

## Integration with Other Commands

On-Call CLI integrates with:
- **incidents** - Route to on-call team members
- **monitors** - Alert on-call responders
- **status-pages** - Display on-call contacts
- **cases** - Assign to on-call personnel

## Workflow Examples

**Complete On-Call Setup:**

```bash
# 1. Create schedule
SCHEDULE_ID=$(dd on-call --action create \
  --name "Backend Support" \
  --team-id team-789 \
  --rotation weekly \
  --members "eng1,eng2,eng3" \
  --timezone "America/New_York" \
  --json | jq -r '.data.id')

# 2. Verify schedule
dd on-call --action get --schedule-id "$SCHEDULE_ID"

# 3. Check who's on-call
dd on-call --action who --schedule-id "$SCHEDULE_ID"
```

**Multi-Region Coverage:**

```bash
# Create schedules for each region
for region in apac emea amer; do
  dd on-call --action create \
    --name "$region Support Schedule" \
    --rotation weekly \
    --members "$region-team-members"
done

# List all regional schedules
dd on-call --action list
```

**Rotation Handoff Automation:**

```bash
#!/bin/bash
# Daily script to check on-call status

echo "=== Current On-Call Status ==="
dd on-call --action who

# Send to Slack/email/notification system
ON_CALL_DATA=$(dd on-call --action who --json)
# Process and notify team
```

## Common Patterns

**Weekend vs Weekday Rotation:**
```bash
# Weekday schedule
dd on-call --action create \
  --name "Weekday Support" \
  --rotation daily \
  --members "weekday-team"

# Weekend schedule
dd on-call --action create \
  --name "Weekend Support" \
  --rotation weekly \
  --members "weekend-team"
```

**Tiered On-Call:**
```bash
# Primary on-call
dd on-call --action create \
  --name "Primary On-Call" \
  --rotation weekly \
  --members "senior-engineers"

# Backup on-call
dd on-call --action create \
  --name "Backup On-Call" \
  --rotation weekly \
  --members "all-engineers"
```

**Schedule Audit:**
```bash
# List all schedules
dd on-call --action list

# Check each schedule status
for schedule in $(dd on-call --action list --json | jq -r '.data[].id'); do
  echo "Schedule: $schedule"
  dd on-call --action who --schedule-id "$schedule"
done
```

## Setup Requirements

**Permissions Required:**
- `on_call_read` - Read on-call schedules
- `on_call_write` - Create/update schedules

**Environment Variables:**
- `DD_API_KEY` - Datadog API key
- `DD_APP_KEY` - Datadog application key

**Team Requirements:**
- Team must be created first
- User IDs must exist in Datadog
- Team members must have appropriate roles

## Troubleshooting

**Schedule creation fails:**
1. Verify team ID exists
2. Check user IDs are valid
3. Ensure timezone format is correct (e.g., "America/New_York")
4. Verify API key has on_call_write permission

**Members not in rotation:**
1. Check user IDs match exactly
2. Verify users are part of the team
3. Ensure rotation start date is set correctly

**Timezone issues:**
1. Use IANA timezone database format
2. Check for typos in timezone string
3. Test with UTC first, then adjust

**Can't see who's on-call:**
1. Schedule may need time to initialize
2. Check schedule has active layers/rotations
3. Verify start date is in the past

## Best Practices

1. **Use Descriptive Names** - Clear schedule names help identification
2. **Document Rotations** - Add descriptions explaining coverage
3. **Test Schedules** - Verify rotations work before production use
4. **Timezone Awareness** - Always specify correct timezone
5. **Regular Reviews** - Update schedules as team changes
6. **Backup Coverage** - Create secondary schedules for redundancy
7. **Automate Checks** - Script who's on-call verification

## Learn More

- [On-Call Documentation](https://docs.datadoghq.com/service_management/on-call/)
- [On-Call Schedules](https://docs.datadoghq.com/service_management/on-call/schedules/)
- [Escalation Policies](https://docs.datadoghq.com/service_management/on-call/escalation_policies/)
- [On-Call Blog Post](https://www.datadoghq.com/blog/datadog-on-call/)

## Related Commands

- `dd incidents` - Incident management and routing
- `dd monitors` - Alert configuration
- `dd status-pages` - Status page management
- `dd cases` - Case management and assignment
