# Git Simulation for 5 Months (Main Branch)

This guide simulates a realistic 5-month development history on `main` using backdated commits.

## Important Risks
- Rewriting `main` history can break clones and PR references.
- Do this only in a private/resume context with no active collaborators.

## Guardrails
- Every commit must map to real code/documentation work.
- Run tests before each milestone commit.
- Keep a backup branch before rewriting history.

## Timeline
- Month 1: security and correctness hardening.
- Month 2: chat/WS performance improvements.
- Month 3: observability and reliability upgrades.
- Month 4: Swagger + architecture docs + ADRs.
- Month 5: CI quality gates and scale-readiness updates.

## Procedure
1. Create backup branch:
   - `git branch backup/pre-simulation`
2. Split current work into milestone chunks.
3. Create commits with explicit dates using environment variables.
4. Verify history and test pass.

## Example (PowerShell)
```powershell
$env:GIT_AUTHOR_DATE="2026-01-08T10:00:00"
$env:GIT_COMMITTER_DATE="2026-01-08T10:00:00"
git commit -m "month1: harden auth and readiness"
```

Repeat with later monthly dates for each milestone.
