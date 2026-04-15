---
name: datadog-backup
description: Backup and restore OpenCode skills to Datadog notebooks. Use when user mentions backup skills, datadog notebook, or restore skills.
compatibility: opencode
metadata:
  tools_needed: "Bash, Read, Write, WebFetch"
  triggers: "backup skills, datadog notebook, restore skills"
---

# Datadog Skill Backup Skill

## Overview
Use this skill to backup OpenCode skills to Datadog notebooks and restore them when needed.

## Tools Required
- **Bash** - Run curl commands, manage-notebooks.sh script
- **Read** - Read skill files
- **Write** - Create notebook content
- **WebFetch** - Check Datadog API docs if needed

## Backup Skills to Notebooks

### Step 1: List current skills
```bash
ls $HOME/rust-town/irclaw/.opencode/skills/
```

### Step 2: Create notebook for each skill
Use the Datadog API to create notebooks containing skill content:

```bash
# Set credentials
export DD_API_KEY="your_api_key"
export DD_APP_KEY="your_app_key"

# Create notebook with skill content
SKILL_NAME="ansible-fleet"
SKILL_CONTENT=$(cat $HOME/rust-town/irclaw/.opencode/skills/${SKILL_NAME}/SKILL.md)

curl -s -X POST \
  -H "DD-API-KEY: ${DD_API_KEY}" \
  -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
  -H "Content-Type: application/json" \
  -d "{
    \"data\": {
      \"type\": \"notebooks\",
      \"attributes\": {
        \"name\": \"Skill: ${SKILL_NAME}\",
        \"cells\": [
          {
            \"type\": \"markdown\",
            \"attributes\": {
              \"definition\": {
                \"text\": \"$(echo "$SKILL_CONTENT" | jq -Rs .)\"
              }
            }
          }
        ]
      }
    }
  }" \
  "https://api.datadoghq.com/api/v1/notebooks"
```

### Step 3: List notebooks to verify
```bash
bash scripts/manage-notebooks.sh --action list
```

## Restore Skills from Notebooks

### Step 1: Get notebook content
```bash
NOTEBOOK_ID=12345
bash scripts/manage-notebooks.sh --action get --notebook-id $NOTEBOOK_ID
```

### Step 2: Extract skill content and save
The markdown cell content can be extracted and saved to a SKILL.md file.

## Reference

- Notebook API: https://docs.datadoghq.com/api/v1/notebooks/
- manage-notebooks.sh location: $HOME/.agents/skills/datadog-operations/scripts/