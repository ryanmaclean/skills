#!/bin/bash
# Manage Datadog SLO Corrections
# Handle SLO corrections for accurate error budget reporting

set -e

# Configuration
DD_SITE=${DD_SITE:-datadoghq.com}
API_BASE="https://api.${DD_SITE}"

# Default values
ACTION="list"
SLO_ID=""
CORRECTION_ID=""
START=""
END=""
CATEGORY="other"
DESCRIPTION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --action)
            ACTION="$2"
            shift 2
            ;;
        --slo-id)
            SLO_ID="$2"
            shift 2
            ;;
        --correction-id)
            CORRECTION_ID="$2"
            shift 2
            ;;
        --start)
            START="$2"
            shift 2
            ;;
        --end)
            END="$2"
            shift 2
            ;;
        --category)
            CATEGORY="$2"
            shift 2
            ;;
        --description)
            DESCRIPTION="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Validate credentials
if [ -z "$DD_API_KEY" ] || [ -z "$DD_APP_KEY" ]; then
    echo '{"status":"error","message":"DD_API_KEY and DD_APP_KEY must be set"}' >&2
    exit 1
fi

# List SLO corrections
list_corrections() {
    local url="${API_BASE}/api/v1/slo/correction"

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${url}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        total_corrections: (.data | length),
        corrections: [.data[] | {
            id: .id,
            slo_id: .attributes.slo_id,
            category: .attributes.category,
            description: .attributes.description,
            start: .attributes.start,
            end: .attributes.end,
            created_at: .attributes.created_at,
            modified_at: .attributes.modified_at
        }]
    }'
}

# Create SLO correction
create_correction() {
    if [ -z "$SLO_ID" ]; then
        echo '{"status":"error","message":"--slo-id is required for create action"}' >&2
        exit 1
    fi

    if [ -z "$START" ] || [ -z "$END" ]; then
        echo '{"status":"error","message":"--start and --end are required for create action"}' >&2
        exit 1
    fi

    # Build payload
    local payload=$(jq -n \
        --arg slo_id "$SLO_ID" \
        --arg category "$CATEGORY" \
        --arg desc "${DESCRIPTION:-SLO correction}" \
        --arg start "$START" \
        --arg end "$END" \
        '{
            data: {
                type: "correction",
                attributes: {
                    slo_id: $slo_id,
                    category: $category,
                    description: $desc,
                    start: ($start | tonumber),
                    end: ($end | tonumber)
                }
            }
        }')

    response=$(curl -s -X POST \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/slo/correction")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        correction: {
            id: .data.id,
            slo_id: .data.attributes.slo_id,
            category: .data.attributes.category,
            description: .data.attributes.description,
            start: .data.attributes.start,
            end: .data.attributes.end
        }
    }'
}

# Get SLO correction
get_correction() {
    if [ -z "$CORRECTION_ID" ]; then
        echo '{"status":"error","message":"--correction-id is required for get action"}' >&2
        exit 1
    fi

    response=$(curl -s \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/slo/correction/${CORRECTION_ID}")

    # Check for API errors
    if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
        echo "$response" | jq '{status: "error", message: .errors[0]}'
        exit 1
    fi

    # Format response
    echo "$response" | jq '{
        status: "success",
        correction: {
            id: .data.id,
            slo_id: .data.attributes.slo_id,
            category: .data.attributes.category,
            description: .data.attributes.description,
            start: .data.attributes.start,
            end: .data.attributes.end,
            created_at: .data.attributes.created_at,
            modified_at: .data.attributes.modified_at,
            creator: .data.attributes.creator
        }
    }'
}

# Delete SLO correction
delete_correction() {
    if [ -z "$CORRECTION_ID" ]; then
        echo '{"status":"error","message":"--correction-id is required for delete action"}' >&2
        exit 1
    fi

    response=$(curl -s -X DELETE \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "DD-APPLICATION-KEY: ${DD_APP_KEY}" \
        "${API_BASE}/api/v1/slo/correction/${CORRECTION_ID}")

    # 204 No Content is success
    if [ -z "$response" ]; then
        echo "{\"status\":\"success\",\"message\":\"SLO correction ${CORRECTION_ID} deleted successfully\"}"
    else
        # Check for errors
        if echo "$response" | jq -e '.errors' > /dev/null 2>&1; then
            echo "$response" | jq '{status: "error", message: .errors[0]}'
            exit 1
        fi
        echo "$response"
    fi
}

# Execute action
case "$ACTION" in
    list)
        list_corrections
        ;;
    create)
        create_correction
        ;;
    get)
        get_correction
        ;;
    delete)
        delete_correction
        ;;
    *)
        echo "{\"status\":\"error\",\"message\":\"Unknown action: $ACTION (use: list, create, get, delete)\"}" >&2
        exit 1
        ;;
esac
