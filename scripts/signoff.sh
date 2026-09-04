#!/usr/bin/env bash
# signoff.sh — local CI. Run the checks on this machine and post the results to
# GitHub as commit statuses, so branch protection can gate on them instead of
# GitHub Actions running the tests.
#
# Three of mappy's four test tiers cannot run on a hosted runner at all:
# integration needs qmk and ZSA's fork, compile needs the ARM toolchain, and
# flash needs a keyboard attached. Signing them off here is the only way they
# gate anything.
#
# Run-and-sign: a context only goes green when its make target exits clean. All
# of them run regardless of earlier failures, so the PR shows exactly which
# check is red. Exits non-zero if any failed.
#
# Requires: gh (authenticated), a Go toolchain, the pinned golangci-lint (mise
# install), qmk with ZSA's fork, and the arm-none-eabi toolchain. Pass --dry-run to preview the statuses without
# running the checks or posting anything.
#
# flash is deliberately not signed. Its tests write no firmware, but they need
# a board plugged in, and a red status because the keyboard is unplugged says
# nothing about the change. Run `make flash-test` by hand when touching flash.
set -uo pipefail

dry=0
[ "${1:-}" = "--dry-run" ] && dry=1

# Sign off on committed, pushed code only — otherwise a green status would
# describe a tree that isn't what's on the PR.
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "✗ working tree has uncommitted changes — commit before signing off." >&2
  exit 1
fi

sha=$(git rev-parse HEAD)
if ! git branch -r --contains "$sha" 2>/dev/null | grep -q .; then
  echo "✗ HEAD (${sha:0:8}) isn't on any remote branch — push before signing off." >&2
  exit 1
fi

repo=$(gh repo view --json nameWithOwner -q .nameWithOwner)

# Fail before running anything rather than after. Posting a status needs push
# access, and finding that out at the end means the checks ran for nothing —
# and, worse, a run that looks green while GitHub was told nothing.
if [ "$dry" = 0 ]; then
  if ! gh api "repos/$repo" --jq .permissions.push | grep -qx true; then
    echo "✗ $(gh api user --jq .login) has no push access to $repo, so statuses cannot be posted." >&2
    echo "  gh auth switch, or set GH_TOKEN to a token for an account that does." >&2
    exit 1
  fi
fi

post() { # context state description
  if [ "$dry" = 1 ]; then
    echo "  [dry-run] would POST status $1=$2 to $repo@${sha:0:8}"
    return 0
  fi
  # A silently dropped post is worse than a red check: the run looks signed
  # off while GitHub knows nothing about it.
  if ! gh api "repos/$repo/statuses/$sha" \
    -f state="$2" -f context="$1" -f description="$3" >/dev/null; then
    echo "  ✗ could not post $1 to GitHub" >&2
    return 1
  fi
}

rc=0
sign() { # context make-target
  echo "▶ $1"
  if [ "$dry" = 1 ]; then post "$1" success "(dry-run)" || rc=1; return; fi

  if make "$2"; then
    post "$1" success "signed off locally by $USER" && echo "  ✔ $1" || rc=1
  else
    post "$1" failure "failed locally" || true
    echo "  ✗ $1"
    rc=1
  fi
}

sign ci/lint         lint
sign ci/unit         unit-test
sign ci/integration  integration-test
sign ci/compile      compile-test

exit $rc
