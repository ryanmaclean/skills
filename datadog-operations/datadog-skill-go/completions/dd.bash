# Bash completion for dd (Datadog CLI)
# Source this file or install to /etc/bash_completion.d/

_dd_completion() {
    local cur prev opts commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # All available commands (22 commands)
    commands="context apm logs metrics security slos watchdog database catalog rum network cicd monitors incidents dashboards workflows synthetics health deploy llm cost version help"

    # Global flags available for all commands
    global_flags="--json --help --version"

    # Command-specific flags
    case "${COMP_WORDS[1]}" in
        context)
            opts="--json --help"
            ;;
        apm)
            opts="--service --from --to --env --json --help"
            ;;
        logs)
            opts="--query --from --to --service --limit --json --help"
            ;;
        metrics)
            opts="--query --from --to --service --json --help"
            ;;
        security)
            opts="--from --to --severity --service --json --help"
            ;;
        slos)
            opts="--slo --service --from --to --json --help"
            ;;
        watchdog)
            opts="--from --to --service --json --help"
            ;;
        database)
            opts="--service --from --to --json --help"
            ;;
        catalog)
            opts="--service --team --json --help"
            ;;
        rum)
            opts="--application --from --to --metrics --json --help"
            ;;
        network)
            opts="--source --dest --from --to --json --help"
            ;;
        cicd)
            opts="--pipeline --from --to --json --help"
            ;;
        monitors)
            opts="--create --update --delete --id --file --json --help"
            ;;
        incidents)
            opts="--from --to --status --service --json --help"
            ;;
        dashboards)
            opts="--list --get --create --update --delete --id --file --json --help"
            ;;
        workflows)
            opts="--list --get --execute --id --json --help"
            ;;
        synthetics)
            opts="--list --get --run --id --json --help"
            ;;
        health)
            opts="--service --from --to --json --help"
            ;;
        deploy)
            opts="--service --json --help"
            ;;
        llm)
            opts="--model --from --to --json --help"
            ;;
        cost)
            opts="--period --service --recommendations --json --help"
            ;;
        version|help)
            opts="--help"
            ;;
        *)
            # If no command yet, suggest commands
            if [[ ${COMP_CWORD} -eq 1 ]]; then
                COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
                return 0
            fi
            ;;
    esac

    # Handle flag completion
    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts} ${global_flags}" -- ${cur}) )
        return 0
    fi

    # Handle specific flag value completion
    case "${prev}" in
        --service)
            # Could query actual services, but for now just suggest
            COMPREPLY=( $(compgen -W "my-service my-api my-worker" -- ${cur}) )
            return 0
            ;;
        --from|--to)
            # Suggest common time ranges
            COMPREPLY=( $(compgen -W "1h 2h 6h 12h 24h 1d 3d 7d 30d" -- ${cur}) )
            return 0
            ;;
        --env)
            COMPREPLY=( $(compgen -W "production staging development test" -- ${cur}) )
            return 0
            ;;
        --severity)
            COMPREPLY=( $(compgen -W "critical high medium low info" -- ${cur}) )
            return 0
            ;;
        --status)
            COMPREPLY=( $(compgen -W "open resolved" -- ${cur}) )
            return 0
            ;;
        --model)
            COMPREPLY=( $(compgen -W "gpt-4 gpt-3.5-turbo claude-3 claude-2 palm-2" -- ${cur}) )
            return 0
            ;;
        --period)
            COMPREPLY=( $(compgen -W "1h 24h 7d 30d" -- ${cur}) )
            return 0
            ;;
        --file)
            # File completion
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
        *)
            ;;
    esac

    # Default: suggest available commands if at position 1
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
    fi
}

# Register completion function
complete -F _dd_completion dd
