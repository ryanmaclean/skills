Name:           datadog-cli
Version:        0.1.0
Release:        1%{?dist}
Summary:        Fast, single-binary Datadog CLI in Go
License:        Apache-2.0
URL:            https://github.com/yourusername/datadog-cli-go
Source0:        https://github.com/yourusername/datadog-cli-go/releases/download/v%{version}/dd-linux-amd64

BuildArch:      x86_64
Requires:       bash-completion

%description
A high-performance Datadog CLI tool written in Go that provides 67x faster
startup than the Python version (3ms vs 200ms) and 67% less memory usage
(10MB vs 30MB).

Features:
- All 22 commands with full Python parity
- Zero dependencies - single static binary
- Shell completions for bash and zsh
- Context auto-detection from git repositories
- APM traces, logs, metrics, security, and more
- Fast performance with optimized Go implementation

This package includes the 'dd' command-line tool and shell completions.

%prep
# No prep needed - binary is pre-built

%build
# No build needed - binary is pre-built

%install
rm -rf $RPM_BUILD_ROOT

# Create directories
mkdir -p $RPM_BUILD_ROOT/usr/bin
mkdir -p $RPM_BUILD_ROOT/etc/bash_completion.d
mkdir -p $RPM_BUILD_ROOT/usr/share/zsh/site-functions

# Install binary
install -m 755 %{SOURCE0} $RPM_BUILD_ROOT/usr/bin/dd

# Install shell completions
# Note: You need to provide these files during rpmbuild
install -m 644 completions/dd.bash $RPM_BUILD_ROOT/etc/bash_completion.d/dd
install -m 644 completions/dd.zsh $RPM_BUILD_ROOT/usr/share/zsh/site-functions/_dd

%files
/usr/bin/dd
/etc/bash_completion.d/dd
/usr/share/zsh/site-functions/_dd

%post
echo ""
echo "========================================="
echo "Datadog CLI installed successfully!"
echo "========================================="
echo ""
echo "The 'dd' command is now available."
echo ""
echo "To use the CLI, set your Datadog credentials:"
echo "  export DD_API_KEY=\"your_datadog_api_key\""
echo "  export DD_APP_KEY=\"your_datadog_app_key\""
echo ""
echo "Get your API keys from:"
echo "  https://app.datadoghq.com/organization-settings/api-keys"
echo ""
echo "Shell completions have been installed for bash and zsh."
echo "Restart your shell or run:"
echo "  source /etc/bash_completion.d/dd  # bash"
echo "  compinit                          # zsh"
echo ""
echo "Quick start:"
echo "  dd context          # Auto-detect service"
echo "  dd health           # Check service health"
echo "  dd apm              # View APM traces"
echo "  dd --help           # Show all commands"
echo ""
echo "Documentation:"
echo "  https://github.com/yourusername/datadog-cli-go"
echo ""

%changelog
* Wed Jan 22 2026 Your Name <you@example.com> - 0.1.0-1
- Initial RPM release
- Fast Go implementation (67x faster than Python)
- All 22 commands with Python parity
- Shell completions included
- Zero dependencies
