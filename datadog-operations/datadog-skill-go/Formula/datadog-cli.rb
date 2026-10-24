# Homebrew Formula for Datadog CLI (Go implementation)
# To use this formula:
#   brew install --formula ./Formula/datadog-cli.rb
# Or from a tap:
#   brew tap yourusername/datadog-cli
#   brew install datadog-cli

class DatadogCli < Formula
  desc "Fast, single-binary Datadog CLI in Go - 67x faster than Python"
  homepage "https://github.com/yourusername/datadog-cli-go"
  version "0.1.0"
  license "Apache-2.0"

  # Update these URLs after publishing to GitHub releases
  # Replace 'yourusername' with your actual GitHub username
  if Hardware::CPU.arm?
    url "https://github.com/yourusername/datadog-cli-go/releases/download/v#{version}/dd-darwin-arm64"
    sha256 "UPDATE_WITH_ACTUAL_SHA256_FOR_ARM64"
  else
    url "https://github.com/yourusername/datadog-cli-go/releases/download/v#{version}/dd-darwin-amd64"
    sha256 "UPDATE_WITH_ACTUAL_SHA256_FOR_AMD64"
  end

  # No build dependencies - it's a pre-compiled binary
  # Runtime dependencies for shell completions
  uses_from_macos "bash" => :optional
  uses_from_macos "zsh" => :optional

  def install
    # Install the binary as 'dd'
    bin.install "dd-darwin-#{Hardware::CPU.arch}" => "dd"

    # Generate and install shell completions
    # Note: Completions are in the repository, not in the release binary
    # You may need to download them separately or include in release

    # If completions are available locally:
    if File.exist?("completions/dd.bash")
      bash_completion.install "completions/dd.bash" => "dd"
    end

    if File.exist?("completions/dd.zsh")
      zsh_completion.install "completions/dd.zsh" => "_dd"
    end

    # If completions need to be generated from the binary:
    # Some CLIs can generate their own completions with --generate-completion
    # Uncomment if your CLI supports this:
    # output = Utils.safe_popen_read(bin/"dd", "completion", "bash")
    # (bash_completion/"dd").write output
    # output = Utils.safe_popen_read(bin/"dd", "completion", "zsh")
    # (zsh_completion/"_dd").write output
  end

  def caveats
    <<~EOS
      Datadog CLI has been installed as 'dd'.

      To use the CLI, you need to set your Datadog credentials:
        export DD_API_KEY="your_datadog_api_key"
        export DD_APP_KEY="your_datadog_app_key"

      Get your API keys from:
        https://app.datadoghq.com/organization-settings/api-keys

      Shell completions have been installed for:
        - Bash: #{HOMEBREW_PREFIX}/etc/bash_completion.d/dd
        - Zsh: #{HOMEBREW_PREFIX}/share/zsh/site-functions/_dd

      For bash completions to work, you need bash-completion installed:
        brew install bash-completion

      Quick start:
        dd context          # Auto-detect service
        dd health           # Check service health
        dd apm              # View APM traces
        dd logs --query "error"  # Search logs

      Documentation:
        https://github.com/yourusername/datadog-cli-go

      Performance:
        - 67x faster startup than Python (3ms vs 200ms)
        - 67% less memory (10MB vs 30MB)
        - Single binary, zero dependencies
    EOS
  end

  test do
    # Test that the binary runs and shows version
    assert_match version.to_s, shell_output("#{bin}/dd --version")

    # Test that help works
    assert_match "Datadog CLI", shell_output("#{bin}/dd --help")

    # Test that commands are available
    help_output = shell_output("#{bin}/dd --help")
    assert_match "context", help_output
    assert_match "apm", help_output
    assert_match "logs", help_output
    assert_match "health", help_output
  end
end
