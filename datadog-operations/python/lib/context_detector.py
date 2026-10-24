"""
Context detector - Auto-discover service context from git, package.json, etc.
Makes Datadog observability "just work" without explicit configuration.
"""

import os
import json
import subprocess
from pathlib import Path
from typing import Optional, Dict, List
from datetime import datetime, timezone
import re


class ServiceContext:
    """Detected service context"""

    def __init__(self):
        self.service_name: Optional[str] = None
        self.repository: Optional[str] = None
        self.current_branch: Optional[str] = None
        self.last_deploy_time: Optional[datetime] = None
        self.last_commit_sha: Optional[str] = None
        self.environment: str = "unknown"
        self.detection_method: Optional[str] = None
        self.confidence: float = 0.0

    def to_dict(self) -> Dict:
        """Convert to dictionary for JSON output"""
        return {
            "service_name": self.service_name,
            "repository": self.repository,
            "current_branch": self.current_branch,
            "last_deploy_time": self.last_deploy_time.isoformat() if self.last_deploy_time else None,
            "last_commit_sha": self.last_commit_sha,
            "environment": self.environment,
            "detection_method": self.detection_method,
            "confidence": self.confidence
        }


class ContextDetector:
    """Auto-detect service context from project files and git"""

    def __init__(self, working_dir: str = "."):
        self.working_dir = Path(working_dir).resolve()

    def detect(self) -> ServiceContext:
        """
        Detect service context using multiple strategies.
        Returns the highest-confidence detection.
        """
        strategies = [
            self._detect_from_git_remote,
            self._detect_from_package_json,
            self._detect_from_docker_compose,
            self._detect_from_directory_name,
        ]

        best_context = None
        best_confidence = 0.0

        for strategy in strategies:
            try:
                context = strategy()
                if context and context.confidence > best_confidence:
                    best_context = context
                    best_confidence = context.confidence
            except Exception:
                continue

        # Enhance with git information
        if best_context:
            self._add_git_metadata(best_context)
            self._detect_environment(best_context)

        return best_context or ServiceContext()

    def _detect_from_git_remote(self) -> Optional[ServiceContext]:
        """Detect service from git remote URL"""
        try:
            result = subprocess.run(
                ["git", "config", "--get", "remote.origin.url"],
                cwd=self.working_dir,
                capture_output=True,
                text=True,
                timeout=5
            )

            if result.returncode != 0:
                return None

            remote_url = result.stdout.strip()

            # Extract repo name from various git URL formats
            # git@github.com:company/repo.git
            # https://github.com/company/repo.git
            patterns = [
                r'[:/]([^/]+)/([^/\.]+)(\.git)?$',  # github/gitlab style
                r'/([^/]+)\.git$',  # simple style
            ]

            for pattern in patterns:
                match = re.search(pattern, remote_url)
                if match:
                    service_name = match.group(2) if match.lastindex >= 2 else match.group(1)

                    context = ServiceContext()
                    context.service_name = service_name.lower()
                    context.repository = remote_url
                    context.detection_method = "git_remote"
                    context.confidence = 0.9
                    return context

        except Exception:
            pass

        return None

    def _detect_from_package_json(self) -> Optional[ServiceContext]:
        """Detect service from package.json name field"""
        package_json = self.working_dir / "package.json"

        if not package_json.exists():
            return None

        try:
            with open(package_json) as f:
                data = json.load(f)

            if "name" in data:
                context = ServiceContext()
                context.service_name = data["name"].lower()
                context.detection_method = "package_json"
                context.confidence = 0.8
                return context

        except Exception:
            pass

        return None

    def _detect_from_docker_compose(self) -> Optional[ServiceContext]:
        """Detect service from docker-compose.yml"""
        compose_files = [
            "docker-compose.yml",
            "docker-compose.yaml",
            "compose.yml",
            "compose.yaml"
        ]

        for filename in compose_files:
            compose_file = self.working_dir / filename
            if compose_file.exists():
                try:
                    with open(compose_file) as f:
                        content = f.read()

                    # Look for service names (simple YAML parsing)
                    # services:
                    #   service-name:
                    match = re.search(r'services:\s*\n\s+(\S+):', content)
                    if match:
                        context = ServiceContext()
                        context.service_name = match.group(1).lower()
                        context.detection_method = "docker_compose"
                        context.confidence = 0.7
                        return context

                except Exception:
                    continue

        return None

    def _detect_from_directory_name(self) -> Optional[ServiceContext]:
        """Fallback: use directory name as service name"""
        dir_name = self.working_dir.name

        # Skip common non-service directory names
        skip_names = {"src", "app", "service", "api", "backend", "frontend", "web"}

        if dir_name.lower() not in skip_names:
            context = ServiceContext()
            context.service_name = dir_name.lower()
            context.detection_method = "directory_name"
            context.confidence = 0.5
            return context

        return None

    def _add_git_metadata(self, context: ServiceContext):
        """Add git branch, commit, and deploy time"""
        try:
            # Current branch
            result = subprocess.run(
                ["git", "rev-parse", "--abbrev-ref", "HEAD"],
                cwd=self.working_dir,
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                context.current_branch = result.stdout.strip()

            # Last commit SHA
            result = subprocess.run(
                ["git", "rev-parse", "HEAD"],
                cwd=self.working_dir,
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                context.last_commit_sha = result.stdout.strip()[:7]

            # Last commit time (as proxy for deploy time)
            result = subprocess.run(
                ["git", "log", "-1", "--format=%ct"],
                cwd=self.working_dir,
                capture_output=True,
                text=True,
                timeout=5
            )
            if result.returncode == 0:
                timestamp = int(result.stdout.strip())
                context.last_deploy_time = datetime.fromtimestamp(timestamp, tz=timezone.utc)

        except Exception:
            pass

    def _detect_environment(self, context: ServiceContext):
        """Detect environment from branch name or environment variables"""
        # Check environment variables
        env_from_var = os.getenv("DD_ENV") or os.getenv("ENVIRONMENT") or os.getenv("NODE_ENV")
        if env_from_var:
            context.environment = env_from_var.lower()
            return

        # Infer from branch name
        if context.current_branch:
            branch = context.current_branch.lower()
            if "main" in branch or "master" in branch:
                context.environment = "production"
            elif "staging" in branch or "stage" in branch:
                context.environment = "staging"
            elif "dev" in branch or "develop" in branch:
                context.environment = "development"
            else:
                context.environment = "development"


def detect_context(working_dir: str = ".") -> ServiceContext:
    """Convenience function to detect context"""
    detector = ContextDetector(working_dir)
    return detector.detect()


if __name__ == "__main__":
    # Test the detector
    context = detect_context()
    print(json.dumps(context.to_dict(), indent=2))
