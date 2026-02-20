#!/usr/bin/env bash
# Set SPEC_DIR to your spec folder, e.g. specs/2026-01-31-example-slug.
# You can also export SPEC_DIR before running this script.
SPEC_DIR="${SPEC_DIR:-specs/YYYY-MM-DD-slug}"

set -euo pipefail
fail() { echo "ERROR: $1" >&2; exit 1; }

# Run this lint when preparing to set status to READY or DONE.
# DRAFT specs may intentionally fail some checks (for example `owners: []`).

get_frontmatter_field() {
  local file="$1"
  local key="$2"

  awk -v key="$key" '
    BEGIN { in_front=0 }
    /^---$/ {
      if (in_front==0) { in_front=1; next }
      exit
    }
    in_front==1 {
      if ($0 ~ ("^[[:space:]]*" key ":[[:space:]]*")) {
        val=$0
        sub("^[[:space:]]*" key ":[[:space:]]*", "", val)
        sub(/[[:space:]]+#.*/, "", val)
        sub(/[[:space:]]*$/, "", val)
        print val
        exit
      }
    }
  ' "$file"
}

get_depends_on_list() {
  local file="$1"

  awk '
    BEGIN { in_front=0; in_dep=0; saw_dep=0 }
    /^---$/ {
      if (in_front==0) { in_front=1; next }
      exit
    }
    in_front==1 {
      if (in_dep==0 && $0 ~ /^[[:space:]]*depends_on:[[:space:]]*\[[^]]*\]([[:space:]]*#.*)?[[:space:]]*$/) {
        saw_dep=1
        val=$0
        sub(/^[[:space:]]*depends_on:[[:space:]]*/, "", val)
        sub(/[[:space:]]+#.*/, "", val)
        gsub(/[[:space:]]/, "", val)
        if (val != "[]") {
          print "ERR:non-empty inline list is not supported; use block list syntax" > "/dev/stderr"
          exit 3
        }
        exit
      }
      if (in_dep==0 && $0 ~ /^[[:space:]]*depends_on:[[:space:]]*(#.*)?$/) {
        saw_dep=1
        in_dep=1
        next
      }
      if (in_dep==1) {
        if ($0 ~ /^[[:space:]]*[a-z_]+:[[:space:]]*/) { exit }
        if ($0 ~ /^[[:space:]]*-[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9-]+([[:space:]]*#.*)?[[:space:]]*$/) {
          dep=$0
          sub(/^[[:space:]]*-[[:space:]]*/, "", dep)
          sub(/[[:space:]]+#.*/, "", dep)
          sub(/[[:space:]]*$/, "", dep)
          print dep
          next
        }
        if ($0 ~ /^[[:space:]]*-[[:space:]]*$/) {
          print "ERR:empty dependency entry" > "/dev/stderr"
          exit 4
        }
        if ($0 ~ /^[[:space:]]*-[[:space:]]*/) {
          print "ERR:invalid dependency format -> " $0 > "/dev/stderr"
          exit 4
        }
        if ($0 ~ /^[[:space:]]*$/) { next }
        if ($0 ~ /^[[:space:]]+/) {
          print "ERR:invalid depends_on line -> " $0 > "/dev/stderr"
          exit 4
        }
      }
    }
    END {
      if (saw_dep==0) {
        print "ERR:depends_on missing in frontmatter" > "/dev/stderr"
        exit 2
      }
    }
  ' "$file"
}

get_missing_links_keys() {
  local file="$1"

  awk '
    function indent_level(s) {
      match(s, /^[[:space:]]*/)
      return RLENGTH
    }
    BEGIN {
      in_front=0
      in_links=0
      saw_links=0
      links_indent=-1
      has_problem=0
      has_requirements=0
      has_design=0
      has_tasks=0
      has_test_plan=0
    }
    /^---$/ {
      if (in_front==0) { in_front=1; next }
      exit
    }
    in_front==1 {
      if (in_links==0 && $0 ~ /^[[:space:]]*links:[[:space:]]*$/) {
        saw_links=1
        in_links=1
        links_indent=indent_level($0)
        next
      }
      if (in_links==1) {
        if ($0 ~ /^[[:space:]]*$/) { next }
        curr_indent=indent_level($0)
        if (curr_indent <= links_indent) {
          in_links=0
        } else if ($0 ~ /^[[:space:]]*[a-z_]+:[[:space:]]*/) {
          line=$0
          sub(/^[[:space:]]*/, "", line)
          key=line
          sub(/:.*/, "", key)
          if (key=="problem") has_problem=1
          else if (key=="requirements") has_requirements=1
          else if (key=="design") has_design=1
          else if (key=="tasks") has_tasks=1
          else if (key=="test_plan") has_test_plan=1
          next
        } else if ($0 ~ /^[[:space:]]*#/) {
          next
        } else {
          print "ERR:invalid links entry -> " $0 > "/dev/stderr"
          exit 3
        }
      }
    }
    END {
      if (saw_links==0) {
        print "ERR:links block missing in frontmatter" > "/dev/stderr"
        exit 2
      }
      missing=""
      if (has_problem==0) missing=missing "problem,"
      if (has_requirements==0) missing=missing "requirements,"
      if (has_design==0) missing=missing "design,"
      if (has_tasks==0) missing=missing "tasks,"
      if (has_test_plan==0) missing=missing "test_plan,"
      sub(/,$/, "", missing)
      print missing
    }
  ' "$file"
}

# 1) Required files (at least Quick mode set)
test -f "$SPEC_DIR/00_problem.md" || fail "missing 00_problem.md"
test -f "$SPEC_DIR/01_requirements.md" || fail "missing 01_requirements.md"
test -f "$SPEC_DIR/03_tasks.md" || fail "missing 03_tasks.md"

# Canonical file for cross-doc consistency
CANON_FILE="$SPEC_DIR/00_problem.md"
CANON_DATE="$(get_frontmatter_field "$CANON_FILE" "spec_date")"
CANON_SLUG="$(get_frontmatter_field "$CANON_FILE" "slug")"
CANON_MODE="$(get_frontmatter_field "$CANON_FILE" "mode")"
CANON_STATUS="$(get_frontmatter_field "$CANON_FILE" "status")"

[ -n "$CANON_DATE" ] || fail "spec_date missing in 00_problem.md"
[ -n "$CANON_SLUG" ] || fail "slug missing in 00_problem.md"
[ -n "$CANON_MODE" ] || fail "mode missing in 00_problem.md"
[ -n "$CANON_STATUS" ] || fail "status missing in 00_problem.md"

if ! printf "%s" "$CANON_DATE" | rg -q "^[0-9]{4}-[0-9]{2}-[0-9]{2}$"; then
  fail "spec_date must be YYYY-MM-DD in 00_problem.md: $CANON_DATE"
fi
if ! printf "%s" "$CANON_SLUG" | rg -q "^[a-z0-9]+(-[a-z0-9]+)*$"; then
  fail "slug must be kebab-case in 00_problem.md: $CANON_SLUG"
fi
if [ "${#CANON_SLUG}" -gt 40 ]; then
  fail "slug too long (>40): $CANON_SLUG"
fi
case "$CANON_MODE" in
  Quick|Full) ;;
  *) fail "unsupported mode value in 00_problem.md: $CANON_MODE (use Quick/Full)" ;;
esac
case "$CANON_STATUS" in
  DRAFT|READY|DONE) ;;
  *) fail "unsupported status value in 00_problem.md: $CANON_STATUS (use DRAFT/READY/DONE)" ;;
esac

# 2) depends_on must exist and be consistent across docs (00_problem.md is canonical)
if rg --files-without-match "^\\s*depends_on:\\s*" "$SPEC_DIR"/*.md >/dev/null; then
  fail "depends_on field missing in one or more docs"
fi
if ! CANON_DEPENDS_ON="$(get_depends_on_list "$CANON_FILE")"; then
  fail "invalid depends_on in 00_problem.md"
fi

for f in "$SPEC_DIR"/*.md; do
  DOC_DATE="$(get_frontmatter_field "$f" "spec_date")"
  DOC_SLUG="$(get_frontmatter_field "$f" "slug")"
  DOC_MODE="$(get_frontmatter_field "$f" "mode")"
  DOC_STATUS="$(get_frontmatter_field "$f" "status")"

  [ "$DOC_DATE" = "$CANON_DATE" ] || fail "spec_date mismatch in $f (expected $CANON_DATE, got $DOC_DATE)"
  [ "$DOC_SLUG" = "$CANON_SLUG" ] || fail "slug mismatch in $f (expected $CANON_SLUG, got $DOC_SLUG)"
  [ "$DOC_MODE" = "$CANON_MODE" ] || fail "mode mismatch in $f (expected $CANON_MODE, got $DOC_MODE)"
  [ "$DOC_STATUS" = "$CANON_STATUS" ] || fail "status mismatch in $f (expected $CANON_STATUS, got $DOC_STATUS)"

  if ! DOC_DEPENDS_ON="$(get_depends_on_list "$f")"; then
    fail "invalid depends_on in $f"
  fi
  if [ "$DOC_DEPENDS_ON" != "$CANON_DEPENDS_ON" ]; then
    fail "depends_on mismatch in $f (must match 00_problem.md)"
  fi
done

# 3) links key set must be present in every doc
for f in "$SPEC_DIR"/*.md; do
  if ! MISSING_LINK_KEYS="$(get_missing_links_keys "$f")"; then
    fail "invalid links block in frontmatter: $f"
  fi
  if [ -n "$MISSING_LINK_KEYS" ]; then
    fail "links key set incomplete in $f (missing: $MISSING_LINK_KEYS)"
  fi
done

# 4) Header placeholders must be gone before READY/DONE
# We use YAML-friendly placeholders in templates:
#   spec_date: null
#   slug: null
#   owners: []
if rg -n "^\\s*spec_date:\\s*null\\s*$|^\\s*slug:\\s*null\\s*$|^\\s*owners:\\s*\\[\\]\\s*$" "$SPEC_DIR"; then
  fail "header placeholders remain (spec_date/slug/owners)"
fi

# 5) Traceability placeholders must be gone
if rg -n "FR-\\?\\?\\?|NFR-\\?\\?\\?|T-\\?\\?\\?|TC-\\?\\?\\?" "$SPEC_DIR"; then
  fail "traceability placeholders remain"
fi

# 6) Required link integrity (no dangling links)
if rg -n "^\\s*design:\\s*02_design\\.md\\s*$" "$SPEC_DIR" >/dev/null; then
  test -f "$SPEC_DIR/02_design.md" || fail "links.design points to missing 02_design.md"
fi
if rg -n "^\\s*test_plan:\\s*04_test_plan\\.md\\s*$" "$SPEC_DIR" >/dev/null; then
  test -f "$SPEC_DIR/04_test_plan.md" || fail "links.test_plan points to missing 04_test_plan.md"
fi

# 7) Mode-specific completeness and link sanity
if [ "$CANON_MODE" = "Full" ]; then
  test -f "$SPEC_DIR/02_design.md" || fail "Full mode requires 02_design.md"
  test -f "$SPEC_DIR/04_test_plan.md" || fail "Full mode requires 04_test_plan.md"
  if rg -n "^\\s*design:\\s*null\\s*$" "$SPEC_DIR"; then
    fail "Full mode must not have links.design: null"
  fi
  if rg -n "^\\s*test_plan:\\s*null\\s*$" "$SPEC_DIR"; then
    fail "Full mode must not have links.test_plan: null"
  fi
fi
if [ "$CANON_MODE" = "Quick" ]; then
  if rg -n "^\\s*design:\\s*02_design\\.md\\s*$" "$SPEC_DIR" >/dev/null; then
    fail "Quick mode must not set links.design: 02_design.md"
  fi
  if rg -n "^\\s*test_plan:\\s*04_test_plan\\.md\\s*$" "$SPEC_DIR" >/dev/null; then
    test -f "$SPEC_DIR/04_test_plan.md" || fail "Quick mode links.test_plan points to missing 04_test_plan.md"
  fi
fi

# 8) Cross-spec dependency gate
CURRENT_SLUG="$(basename "$SPEC_DIR")"
if [ -n "$CANON_DEPENDS_ON" ]; then
  DUP_DEP="$(printf "%s\n" "$CANON_DEPENDS_ON" | sort | uniq -d | head -n 1)"
  [ -z "$DUP_DEP" ] || fail "depends_on has duplicate entry: $DUP_DEP"
fi

while IFS= read -r DEP_SLUG; do
  [ -n "$DEP_SLUG" ] || continue

  if [ "$DEP_SLUG" = "$CURRENT_SLUG" ]; then
    fail "depends_on must not include current spec: $DEP_SLUG"
  fi

  DEP_DIR="specs/$DEP_SLUG"
  test -d "$DEP_DIR" || fail "depends_on points to missing folder: $DEP_DIR"
  if ! compgen -G "$DEP_DIR/*.md" >/dev/null; then
    fail "dependency folder has no markdown spec docs: $DEP_DIR"
  fi
  for dep_file in "$DEP_DIR"/*.md; do
    DEP_STATUS="$(get_frontmatter_field "$dep_file" "status")"
    case "$DEP_STATUS" in
      DONE) ;;
      DRAFT|READY)
        fail "dependency not DONE yet: $dep_file"
        ;;
      "")
        fail "dependency missing status in frontmatter: $dep_file"
        ;;
      *)
        fail "dependency has unsupported status in frontmatter: $dep_file ($DEP_STATUS)"
        ;;
    esac
  done
done <<< "$CANON_DEPENDS_ON"
