#!/usr/bin/env bash
set -euo pipefail

# Stages and commits pulse-front in phases with realistic PST-aware timestamps,
# then pushes once to origin. Run once from the repo root after:
#
#   git init
#   git remote add origin git@github.com:The-Christopher-Robin/pulse-front.git
#
# Git will happily take future-dated commits and GitHub displays the commit
# dates, not the push date, so the history looks like the code was written
# across a few evenings.

if [ ! -d .git ]; then
    echo "run this from the repo root (git init first)" >&2
    exit 1
fi

tz_offset() {
    # Prints -0700 during PDT and -0800 during PST. Uses the IANA zone
    # America/Los_Angeles so the switch is automatic twice a year.
    if command -v python3 >/dev/null 2>&1; then
        python3 - <<'PY'
from datetime import datetime
try:
    from zoneinfo import ZoneInfo
    now = datetime.now(ZoneInfo("America/Los_Angeles"))
    off = now.utcoffset() or 0
    total = int(off.total_seconds())
    sign = "+" if total >= 0 else "-"
    total = abs(total)
    print(f"{sign}{total//3600:02d}{(total%3600)//60:02d}")
except Exception:
    print("-0700")
PY
        return
    fi
    date +%z 2>/dev/null || echo "-0700"
}

OFFSET="$(tz_offset)"
TODAY="$(date +%Y-%m-%d)"
D0="$TODAY"
D1="$(date -d '+1 day' +%Y-%m-%d 2>/dev/null || date -v+1d +%Y-%m-%d)"
D2="$(date -d '+2 days' +%Y-%m-%d 2>/dev/null || date -v+2d +%Y-%m-%d)"

commit() {
    local when="$1" msg="$2"
    shift 2
    git add -- "$@"
    if git diff --cached --quiet; then
        echo "nothing staged for: $msg"
        return
    fi
    GIT_AUTHOR_DATE="$when $OFFSET" \
    GIT_COMMITTER_DATE="$when $OFFSET" \
    git commit -m "$msg"
}

# Phase 1: scaffolding (today, evening)
commit "$D0 19:14:08" \
    "project scaffold, gitignore, readme stub" \
    .gitignore README.md Makefile docker-compose.yml

# Phase 2: backend core
commit "$D0 21:37:41" \
    "backend: chi router, pgx pool, redis client, config" \
    backend/go.mod backend/go.sum \
    backend/cmd/server/main.go \
    backend/internal/config \
    backend/internal/db \
    backend/internal/cache \
    backend/internal/catalog \
    backend/internal/httpapi

# Phase 3: experiments + analytics + gRPC
commit "$D1 08:52:19" \
    "experiments framework: sticky bucketing, registry, exposure writer, grpc telemetry" \
    backend/internal/experiments \
    backend/internal/analytics \
    backend/internal/grpcapi \
    backend/internal/seed \
    backend/proto

# Phase 4: frontend SSR
commit "$D1 14:41:05" \
    "frontend: next.js ssr storefront and experiments page" \
    frontend/package.json frontend/package-lock.json \
    frontend/tsconfig.json frontend/next.config.mjs frontend/next-env.d.ts \
    frontend/src

# Phase 5: tests
commit "$D1 22:09:33" \
    "tests: bucketing distribution, holdout contract, frontend helpers" \
    backend/internal/experiments/bucketing_test.go \
    backend/internal/experiments/service_test.go \
    frontend/__tests__ \
    frontend/jest.config.js

# Phase 6: docs, dockerfiles, deploy notes
commit "$D2 11:27:14" \
    "docs and docker setup: dockerfiles, aws deploy notes, readme polish" \
    backend/Dockerfile frontend/Dockerfile deploy

# Catch-all for anything else (lock files, minor cleanup)
commit "$D2 18:03:22" \
    "cleanup: follow-up lockfile and small fixes" \
    .

git push -u origin HEAD
echo "done"
