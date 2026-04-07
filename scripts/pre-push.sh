#!/bin/bash
# pre-push hook: Run LLM-Judge eval when skill-related files change.
# Installed by `ina setup`. Requires Go and claude CLI.

set -e

REMOTE="$1"
URL="$2"

# Determine base branch
BASE_BRANCH="origin/main"
if ! git rev-parse --verify "$BASE_BRANCH" >/dev/null 2>&1; then
    BASE_BRANCH="origin/master"
fi

# Get changed files
CHANGED=$(git diff --name-only "$BASE_BRANCH"...HEAD 2>/dev/null || git diff --name-only HEAD~1 HEAD 2>/dev/null || echo "")

if [ -z "$CHANGED" ]; then
    exit 0
fi

# Check if skill-related files changed
EVAL_TRIGGER=$(echo "$CHANGED" | grep -E '^(skills/|test/eval|test/fixtures/)' || true)

if [ -z "$EVAL_TRIGGER" ]; then
    exit 0
fi

# Extract changed skill names
EVAL_SKILLS=$(echo "$CHANGED" | grep '^skills/' | cut -d'/' -f2 | sort -u | paste -sd',' -)

# Check prerequisites
if ! command -v go >/dev/null 2>&1; then
    echo "[ina eval] WARNING: Go not found. Skipping LLM-Judge eval."
    echo "[ina eval] Install Go to enable pre-push skill evaluation."
    exit 0
fi

if ! command -v claude >/dev/null 2>&1; then
    echo "[ina eval] WARNING: claude CLI not found. Skipping LLM-Judge eval."
    exit 0
fi

if [ -z "$EVAL_SKILLS" ]; then
    EVAL_SKILLS="all"
fi

echo "[ina eval] Skill changes detected: $EVAL_SKILLS"
echo "[ina eval] Running LLM-Judge eval (Tier 3)..."

# Find project root
PROJECT_ROOT=$(git rev-parse --show-toplevel)

cd "$PROJECT_ROOT"
EVAL_SKILLS="$EVAL_SKILLS" INFA_EVAL=1 go test ./test/ -run TestSkillEval -v -timeout 600s

echo "[ina eval] All evaluations passed."
