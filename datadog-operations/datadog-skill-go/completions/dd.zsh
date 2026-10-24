#compdef dd
# Zsh completion for dd (Datadog CLI)
# Install to /usr/local/share/zsh/site-functions/_dd or add to fpath

_dd() {
    local line state

    _arguments -C \
        '1: :->command' \
        '*:: :->args'

    case $state in
        command)
            local -a commands
            commands=(
                'context:Auto-detect service from git repository'
                'apm:Query APM traces and performance metrics'
                'logs:Search and analyze logs'
                'metrics:Query time-series metrics'
                'security:View security signals and events'
                'slos:Monitor Service Level Objectives'
                'watchdog:Detect anomalies with Watchdog'
                'database:Monitor database performance'
                'catalog:Browse service catalog'
                'rum:Real User Monitoring analytics'
                'network:Network performance monitoring'
                'cicd:CI/CD Visibility metrics'
                'monitors:Manage monitors (create, update, delete)'
                'incidents:Incident management'
                'dashboards:Manage dashboards'
                'workflows:Workflow automation'
                'synthetics:Synthetic test management'
                'health:Multi-signal health check'
                'deploy:Deployment readiness check'
                'llm:LLM observability and cost tracking'
                'cost:Infrastructure cost analysis'
                'version:Show version information'
                'help:Show help information'
            )
            _describe 'command' commands
            ;;
        args)
            case $line[1] in
                context)
                    _arguments \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                apm)
                    _arguments \
                        '--service[Service name]:service:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--env[Environment]:env:(production staging development test)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                logs)
                    _arguments \
                        '--query[Search query]:query:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--service[Service name]:service:' \
                        '--limit[Number of results]:limit:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                metrics)
                    _arguments \
                        '--query[Metric query]:query:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--service[Service name]:service:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                security)
                    _arguments \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--severity[Severity level]:severity:(critical high medium low info)' \
                        '--service[Service name]:service:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                slos)
                    _arguments \
                        '--slo[SLO name]:slo:' \
                        '--service[Service name]:service:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                watchdog)
                    _arguments \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--service[Service name]:service:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                database)
                    _arguments \
                        '--service[Service name]:service:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                catalog)
                    _arguments \
                        '--service[Service name]:service:' \
                        '--team[Team name]:team:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                rum)
                    _arguments \
                        '--application[Application name]:application:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--metrics[Metrics to show]:metrics:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                network)
                    _arguments \
                        '--source[Source]:source:' \
                        '--dest[Destination]:dest:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                cicd)
                    _arguments \
                        '--pipeline[Pipeline name]:pipeline:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                monitors)
                    _arguments \
                        '--create[Create monitor]' \
                        '--update[Update monitor]' \
                        '--delete[Delete monitor]' \
                        '--id[Monitor ID]:id:' \
                        '--file[Monitor definition file]:file:_files' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                incidents)
                    _arguments \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--status[Status]:status:(open resolved)' \
                        '--service[Service name]:service:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                dashboards)
                    _arguments \
                        '--list[List dashboards]' \
                        '--get[Get dashboard]' \
                        '--create[Create dashboard]' \
                        '--update[Update dashboard]' \
                        '--delete[Delete dashboard]' \
                        '--id[Dashboard ID]:id:' \
                        '--file[Dashboard definition file]:file:_files' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                workflows)
                    _arguments \
                        '--list[List workflows]' \
                        '--get[Get workflow]' \
                        '--execute[Execute workflow]' \
                        '--id[Workflow ID]:id:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                synthetics)
                    _arguments \
                        '--list[List synthetic tests]' \
                        '--get[Get synthetic test]' \
                        '--run[Run synthetic test]' \
                        '--id[Test ID]:id:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                health)
                    _arguments \
                        '--service[Service name]:service:' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                deploy)
                    _arguments \
                        '--service[Service name]:service:' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                llm)
                    _arguments \
                        '--model[Model name]:model:(gpt-4 gpt-3.5-turbo claude-3 claude-2 palm-2)' \
                        '--from[Start time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--to[End time]:time:(1h 2h 6h 12h 24h 1d 3d 7d 30d)' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                cost)
                    _arguments \
                        '--period[Time period]:period:(1h 24h 7d 30d)' \
                        '--service[Service name]:service:' \
                        '--recommendations[Show recommendations]' \
                        '--json[Output in JSON format]' \
                        '--help[Show help]'
                    ;;
                version|help)
                    _arguments '--help[Show help]'
                    ;;
            esac
            ;;
    esac
}

_dd "$@"
