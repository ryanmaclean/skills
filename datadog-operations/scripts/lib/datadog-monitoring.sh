#!/bin/bash
# Datadog Monitoring Library
# Provides functions to send traces, metrics, and logs to Datadog
# Compatible with bash 3.2+ (macOS compatible)

# Configuration variables (set these in your environment)
# DD_MONITORING_ENABLED: Set to "true" to enable monitoring (default: false)
# DD_MONITORING_SERVICE: Service name for monitoring (default: "datadog-skill")
# DD_API_KEY: Datadog API key (required for logs and metrics)
# DD_SITE: Datadog site (default: datadoghq.com)
# DD_AGENT_HOST: Datadog agent host for traces (default: localhost)
# DD_TRACE_AGENT_PORT: Datadog trace agent port (default: 8126)

# Bash 3.2 compatible associative array workaround
declare -a _OPERATION_STACK_NAMES=()
declare -a _OPERATION_STACK_TIMES=()

# Check if monitoring is enabled
_is_monitoring_enabled() {
    [ "${DD_MONITORING_ENABLED:-false}" = "true" ]
}

# Get current timestamp in milliseconds
_get_timestamp_ms() {
    # macOS compatible - use date command
    local ts_s=$(date +%s)
    echo $((ts_s * 1000))
}

# Get current timestamp in nanoseconds (for traces)
_get_timestamp_ns() {
    # Convert milliseconds to nanoseconds
    local ts_ms=$(_get_timestamp_ms)
    echo $((ts_ms * 1000000))
}

# Generate random trace/span ID
_generate_id() {
    # Generate 64-bit random number
    printf "%d" $((RANDOM * RANDOM * RANDOM))
}

# URL encode function for bash 3.2
_urlencode() {
    local string="$1"
    local strlen=${#string}
    local encoded=""
    local pos c o

    for (( pos=0 ; pos<strlen ; pos++ )); do
        c=${string:$pos:1}
        case "$c" in
            [-_.~a-zA-Z0-9] )
                o="${c}"
                ;;
            * )
                printf -v o '%%%02x' "'$c"
                ;;
        esac
        encoded="${encoded}${o}"
    done
    echo "${encoded}"
}

# Build tags string from array (bash 3.2 compatible)
_build_tags() {
    local result=""
    local first=true

    for tag in "$@"; do
        if [ "$first" = true ]; then
            result="$tag"
            first=false
        else
            result="$result,$tag"
        fi
    done

    echo "$result"
}

# Send trace to Datadog agent
# Usage: send_trace "script_name" "operation" 1234 "ok" "tag1:value1" "tag2:value2"
send_trace() {
    if ! _is_monitoring_enabled; then
        return 0
    fi

    local script_name="$1"
    local operation="$2"
    local duration_ms="$3"
    local status="${4:-ok}"
    shift 4
    local tags=("$@")

    local agent_host="${DD_AGENT_HOST:-localhost}"
    local agent_port="${DD_TRACE_AGENT_PORT:-8126}"
    local service="${DD_MONITORING_SERVICE:-datadog-skill}"
    local env="${DD_ENV:-production}"

    local trace_id=$(_generate_id)
    local span_id=$(_generate_id)
    local current_ns=$(_get_timestamp_ns)
    local duration_ns=$((duration_ms * 1000000))
    local start_time=$((current_ns - duration_ns))

    # Build meta tags
    local meta_tags=""
    for tag in "${tags[@]}"; do
        local key="${tag%%:*}"
        local value="${tag#*:}"
        if [ -z "$meta_tags" ]; then
            meta_tags="\"$key\": \"$value\""
        else
            meta_tags="$meta_tags, \"$key\": \"$value\""
        fi
    done

    # Set error flag if status is not ok
    local error_flag=0
    if [ "$status" != "ok" ]; then
        error_flag=1
    fi

    # Build trace payload (Datadog APM format v0.4)
    local payload="[[{
        \"trace_id\": $trace_id,
        \"span_id\": $span_id,
        \"name\": \"$operation\",
        \"resource\": \"$script_name\",
        \"service\": \"$service\",
        \"type\": \"script\",
        \"start\": $start_time,
        \"duration\": $duration_ns,
        \"error\": $error_flag,
        \"meta\": {
            \"env\": \"$env\",
            \"script.name\": \"$script_name\",
            \"operation.status\": \"$status\",
            $meta_tags
        },
        \"metrics\": {
            \"_sampling_priority_v1\": 1
        }
    }]]"

    # Send to agent (fail silently)
    curl -s -X PUT \
        "http://${agent_host}:${agent_port}/v0.4/traces" \
        -H "Content-Type: application/json" \
        -H "Datadog-Meta-Tracer-Version: custom-bash-1.0" \
        -d "$payload" \
        >/dev/null 2>&1 || true

    return 0
}

# Send metric to Datadog API
# Usage: send_metric "metric.name" 42 "tag1:value1" "tag2:value2"
send_metric() {
    if ! _is_monitoring_enabled; then
        return 0
    fi

    local metric_name="$1"
    local value="$2"
    shift 2
    local tags=("$@")

    if [ -z "$DD_API_KEY" ]; then
        echo "[WARN] DD_API_KEY not set, cannot send metric" >&2
        return 1
    fi

    local site="${DD_SITE:-datadoghq.com}"
    local host="${DD_HOSTNAME:-$(hostname)}"
    local service="${DD_MONITORING_SERVICE:-datadog-skill}"
    local timestamp=$(date +%s)

    # Add default tags
    local all_tags=("service:$service" "${tags[@]}")
    local tags_json=""

    for tag in "${all_tags[@]}"; do
        if [ -z "$tags_json" ]; then
            tags_json="\"$tag\""
        else
            tags_json="$tags_json, \"$tag\""
        fi
    done

    # Build metrics payload
    local payload="{
        \"series\": [{
            \"metric\": \"$metric_name\",
            \"points\": [[$(date +%s), $value]],
            \"type\": \"gauge\",
            \"host\": \"$host\",
            \"tags\": [$tags_json]
        }]
    }"

    # Send to Datadog API (fail silently)
    curl -s -X POST \
        "https://api.${site}/api/v1/series" \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        >/dev/null 2>&1 || true

    return 0
}

# Send log to Datadog API
# Usage: send_log "info" "Message text" "tag1:value1" "tag2:value2"
send_log() {
    if ! _is_monitoring_enabled; then
        return 0
    fi

    local level="$1"
    local message="$2"
    shift 2
    local tags=("$@")

    if [ -z "$DD_API_KEY" ]; then
        echo "[WARN] DD_API_KEY not set, cannot send log" >&2
        return 1
    fi

    local site="${DD_SITE:-datadoghq.com}"
    local host="${DD_HOSTNAME:-$(hostname)}"
    local service="${DD_MONITORING_SERVICE:-datadog-skill}"
    local timestamp=$(_get_timestamp_ms)

    # Map level to Datadog status
    local status="info"
    case "$level" in
        error|ERROR)
            status="error"
            ;;
        warn|WARNING|WARN)
            status="warn"
            ;;
        info|INFO)
            status="info"
            ;;
        debug|DEBUG)
            status="debug"
            ;;
    esac

    # Build tags string
    local tags_str="service:$service"
    for tag in "${tags[@]}"; do
        tags_str="$tags_str,$tag"
    done

    # Escape message for JSON
    local escaped_message="${message//\\/\\\\}"
    escaped_message="${escaped_message//\"/\\\"}"
    escaped_message="${escaped_message//$'\n'/\\n}"
    escaped_message="${escaped_message//$'\r'/\\r}"
    escaped_message="${escaped_message//$'\t'/\\t}"

    # Build log payload
    local payload="[{
        \"ddsource\": \"bash-script\",
        \"ddtags\": \"$tags_str\",
        \"hostname\": \"$host\",
        \"message\": \"$escaped_message\",
        \"service\": \"$service\",
        \"status\": \"$status\",
        \"timestamp\": $timestamp
    }]"

    # Send to Datadog API (fail silently)
    curl -s -X POST \
        "https://http-intake.logs.${site}/api/v2/logs" \
        -H "DD-API-KEY: ${DD_API_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        >/dev/null 2>&1 || true

    return 0
}

# Start an operation (for timing)
# Usage: start_operation "operation_name"
start_operation() {
    local operation_name="$1"
    local start_time=$(_get_timestamp_ms)

    # Push to stack (bash 3.2 compatible)
    _OPERATION_STACK_NAMES+=("$operation_name")
    _OPERATION_STACK_TIMES+=("$start_time")

    return 0
}

# End an operation and send trace
# Usage: end_operation "ok" "tag1:value1" "tag2:value2"
end_operation() {
    local status="${1:-ok}"
    shift
    local tags=("$@")

    # Check if stack is empty
    if [ ${#_OPERATION_STACK_NAMES[@]} -eq 0 ]; then
        echo "[WARN] end_operation called without matching start_operation" >&2
        return 1
    fi

    # Pop from stack (bash 3.2 compatible)
    local last_idx=$((${#_OPERATION_STACK_NAMES[@]} - 1))
    local operation_name="${_OPERATION_STACK_NAMES[$last_idx]}"
    local start_time="${_OPERATION_STACK_TIMES[$last_idx]}"

    unset "_OPERATION_STACK_NAMES[$last_idx]"
    unset "_OPERATION_STACK_TIMES[$last_idx]"
    _OPERATION_STACK_NAMES=("${_OPERATION_STACK_NAMES[@]}")
    _OPERATION_STACK_TIMES=("${_OPERATION_STACK_TIMES[@]}")

    # Calculate duration
    local end_time=$(_get_timestamp_ms)
    local duration=$((end_time - start_time))

    # Get script name from caller
    local script_name="${BASH_SOURCE[1]##*/}"

    # Send trace
    send_trace "$script_name" "$operation_name" "$duration" "$status" "${tags[@]}"

    return 0
}

# Wrap a command with monitoring
# Usage: monitor_command "operation_name" command args...
monitor_command() {
    local operation_name="$1"
    shift
    local command=("$@")

    start_operation "$operation_name"

    local exit_code=0
    if "${command[@]}"; then
        end_operation "ok"
    else
        exit_code=$?
        end_operation "error" "exit_code:$exit_code"
        return $exit_code
    fi

    return 0
}

# Send script execution metrics
# Usage: send_script_metrics "script_name" duration_ms exit_code
send_script_metrics() {
    if ! _is_monitoring_enabled; then
        return 0
    fi

    local script_name="$1"
    local duration_ms="$2"
    local exit_code="${3:-0}"

    local status="success"
    if [ "$exit_code" -ne 0 ]; then
        status="failure"
    fi

    # Send duration metric
    send_metric "datadog.skill.script.duration" "$duration_ms" \
        "script:$script_name" "status:$status"

    # Send execution count
    send_metric "datadog.skill.script.executions" 1 \
        "script:$script_name" "status:$status"

    return 0
}

# Initialize monitoring for a script
# Usage: init_monitoring "script_name"
# Call this at the start of your script and use trap to ensure cleanup
init_monitoring() {
    local script_name="$1"

    if ! _is_monitoring_enabled; then
        return 0
    fi

    # Set global start time
    export _SCRIPT_START_TIME=$(_get_timestamp_ms)
    export _SCRIPT_NAME="$script_name"

    # Send start log
    send_log "info" "Script execution started: $script_name" \
        "script:$script_name" "event:start"

    return 0
}

# Finalize monitoring for a script
# Usage: finalize_monitoring [exit_code]
# Call this at the end of your script or in a trap
finalize_monitoring() {
    local exit_code="${1:-0}"

    if ! _is_monitoring_enabled; then
        return 0
    fi

    if [ -z "$_SCRIPT_START_TIME" ] || [ -z "$_SCRIPT_NAME" ]; then
        return 0
    fi

    # Calculate total duration
    local end_time=$(_get_timestamp_ms)
    local duration=$(( end_time - _SCRIPT_START_TIME ))

    # Determine status
    local status="ok"
    local log_level="info"
    if [ "$exit_code" -ne 0 ]; then
        status="error"
        log_level="error"
    fi

    # Send metrics
    send_script_metrics "$_SCRIPT_NAME" "$duration" "$exit_code"

    # Send trace
    send_trace "$_SCRIPT_NAME" "script.execution" "$duration" "$status" \
        "exit_code:$exit_code"

    # Send completion log
    send_log "$log_level" "Script execution completed: $_SCRIPT_NAME (duration: ${duration}ms, exit: $exit_code)" \
        "script:$_SCRIPT_NAME" "event:complete" "exit_code:$exit_code" "duration_ms:$duration"

    return 0
}

# Print monitoring configuration (for debugging)
print_monitoring_config() {
    cat <<EOF
Datadog Monitoring Configuration:
  Enabled: ${DD_MONITORING_ENABLED:-false}
  Service: ${DD_MONITORING_SERVICE:-datadog-skill}
  Site: ${DD_SITE:-datadoghq.com}
  Agent Host: ${DD_AGENT_HOST:-localhost}
  Agent Port: ${DD_TRACE_AGENT_PORT:-8126}
  API Key: $([ -n "$DD_API_KEY" ] && echo "set" || echo "not set")
  Hostname: ${DD_HOSTNAME:-$(hostname)}
  Environment: ${DD_ENV:-not set}
EOF
}
