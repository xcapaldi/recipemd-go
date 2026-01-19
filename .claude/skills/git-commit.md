---
name: git-commit
description: Commit code to git
---

# Conventional Commits 

When comitting code always use conventional commits.

## Format
```
type(scope): description

[optional body]

[optional footer]
```

## Types
- `feat` - new feature
- `fix` - bug fix
- `chore` - documentation, code refactoring, build tooling, CI, standalone testing, etc.

## Scope
Use scope to indicate the code section (package, module, file):
- `feat(cmd/cli): add user endpoint`
- `fix(internal/auth): prevent token expiry edge case`
- `chore(pkg/utils): simplify string helpers`

## Breaking Changes
Add `!` after type/scope for breaking changes:
- `feat(cmd/cli)!: change response format to JSON:API`
- `chore!: rename exported functions`
