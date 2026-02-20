#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

ok() {
  echo "OK: $*"
}

require_file() {
  local path="$1"
  [ -f "$path" ] || fail "missing file: $path"
}

require_pattern() {
  local pattern="$1"
  rg -Fq "$pattern" AGENTS.md || fail "AGENTS.md missing required rule: $pattern"
}

require_file "AGENTS.md"
require_file "scripts/spec-lint.sh"
[ -x "scripts/spec-lint.sh" ] || fail "scripts/spec-lint.sh is not executable"

require_file "assets/00_problem_template.md"
require_file "assets/01_requirements_template.md"
require_file "assets/02_design_template.md"
require_file "assets/03_tasks_template.md"
require_file "assets/04_test_plan_template.md"

for template in assets/*_template.md; do
  head -n 1 "$template" | rg -q '^---$' || fail "template must start with YAML frontmatter: $template"
done
ok "spec templates and spec-lint script are present"

require_pattern 'SPEC_DIR="specs/YYYY-MM-DD-slug" bash scripts/spec-lint.sh'
require_pattern "Domain must not import"
require_pattern "Application must not import"
require_pattern '`go env GOMOD`'
require_pattern '`go env GOWORK`'
require_pattern "go list ./..."
require_pattern "go test -short ./..."
require_pattern "go vet ./..."
require_pattern 'Do not introduce `src/`'
ok "AGENTS.md contains critical policy rules"

SPEC_DIR_INPUT="${SPEC_DIR:-${1:-}}"
if [ -z "$SPEC_DIR_INPUT" ]; then
  SPEC_DIR_INPUT="$(find specs -mindepth 1 -maxdepth 1 -type d | sort | tail -n 1 || true)"
fi
[ -n "$SPEC_DIR_INPUT" ] || fail "no spec directory found; pass one explicitly: scripts/verify-agents.sh specs/YYYY-MM-DD-slug"
[ -d "$SPEC_DIR_INPUT" ] || fail "SPEC_DIR does not exist: $SPEC_DIR_INPUT"

require_file "$SPEC_DIR_INPUT/00_problem.md"
require_file "$SPEC_DIR_INPUT/01_requirements.md"
require_file "$SPEC_DIR_INPUT/03_tasks.md"

SPEC_DIR="$SPEC_DIR_INPUT" bash scripts/spec-lint.sh >/tmp/verify-agents-spec-lint.out 2>&1 || {
  cat /tmp/verify-agents-spec-lint.out >&2
  fail "spec-lint failed for $SPEC_DIR_INPUT"
}
ok "spec-lint passed for $SPEC_DIR_INPUT"

go env GOMOD >/dev/null
go env GOWORK >/dev/null
go list ./... >/dev/null
ok "go env and go list checks passed"

echo "ALL CHECKS PASSED"
