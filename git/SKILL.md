# Skill: Git Version Control

## Overview

Git is a distributed version control system. This skill covers common Git workflows, branching strategies, and best practices for collaborating on code.

## Key Concepts

- **Repository**: A directory tracked by Git, containing a full history of changes.
- **Commit**: A snapshot of changes with a message, author, and timestamp.
- **Branch**: A lightweight movable pointer to a commit, used to isolate work.
- **Remote**: A hosted copy of a repository (e.g., on GitHub).
- **Pull Request (PR)**: A request to merge changes from one branch into another, typically reviewed by peers.
- **Merge / Rebase**: Two strategies for integrating changes from one branch into another.

## Common Tasks

### Clone a repository
```bash
git clone https://github.com/<owner>/<repo>.git
```

### Create and switch to a new branch
```bash
git checkout -b feature/my-feature
# or (Git 2.23+)
git switch -c feature/my-feature
```

### Stage and commit changes
```bash
git add .
git commit -m "feat: describe your change"
```

### Push a branch to a remote
```bash
git push -u origin feature/my-feature
```

### Pull latest changes from the default branch
```bash
git fetch origin
git rebase origin/main
```

### Squash commits before merging
```bash
git rebase -i origin/main
# Mark all but the first commit as "squash" or "s"
```

### Undo the last commit (keep changes staged)
```bash
git reset --soft HEAD~1
```

### View commit history
```bash
git log --oneline --graph --decorate
```

## Best Practices

- Write clear, imperative commit messages (e.g., `fix: correct off-by-one error`).
- Keep branches short-lived and focused on a single concern.
- Rebase feature branches on the latest `main` before opening a PR to reduce merge conflicts.
- Never force-push to shared branches (`main`, `develop`).
- Use `.gitignore` to exclude build artifacts, secrets, and editor-specific files.
- Sign commits with GPG where required by your organization.

## References

- [Pro Git Book](https://git-scm.com/book/en/v2)
- [GitHub Docs – Pull Requests](https://docs.github.com/en/pull-requests)
- [Conventional Commits](https://www.conventionalcommits.org/)
