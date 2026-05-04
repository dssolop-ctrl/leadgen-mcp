#!/usr/bin/env bash
# lock.sh — per-city RUNNING.lock helper.
#
# Subcommands:
#   acquire <city> <run_id> [phase]   → exit 0 if acquired, 1 if active lock exists, 2 if stale recovered
#   release <city> [run_id]           → remove lock (only if matches run_id)
#   update_phase <city> <run_id> <phase>
#   inspect <city>                    → print current lock content (or empty)
#
# Lock format (YAML):
#   city: <city>
#   run_id: <run_id>
#   started_at: <ISO8601>
#   phase: <phase>
#   ttl_minutes: 120
#   pid_hint: <PID> (informational, not enforced)
#
# TTL: from caps_defaults.yaml.running_lock_ttl_minutes (default 120).
# If lock older than TTL → considered stale; acquire recovers it (returns 2).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_ROOT="${SCRIPT_DIR}/../runtime"

TTL_MINUTES="${AUTOPILOT_LOCK_TTL_MINUTES:-120}"

now_iso() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }
now_epoch() { date -u +%s; }

iso_to_epoch() {
  # Best-effort: Git Bash on Windows + GNU date both support -d / +%s
  date -u -d "$1" +%s 2>/dev/null || echo 0
}

lock_path() {
  local city="$1"
  echo "${RUNTIME_ROOT}/${city}/locks/RUNNING.lock"
}

cmd="${1:-}"
shift || true

case "$cmd" in
  acquire)
    city="${1:?city required}"
    run_id="${2:?run_id required}"
    phase="${3:-started}"

    lock="$(lock_path "$city")"
    mkdir -p "$(dirname "$lock")"

    if [[ -f "$lock" ]]; then
      started_at="$(grep -E '^started_at:' "$lock" | sed 's/^started_at: //; s/^"//; s/"$//')"
      existing_run="$(grep -E '^run_id:' "$lock" | sed 's/^run_id: //; s/^"//; s/"$//')"
      lock_epoch="$(iso_to_epoch "$started_at")"
      now=$(now_epoch)
      age_minutes=$(( (now - lock_epoch) / 60 ))

      if (( lock_epoch > 0 )) && (( age_minutes < TTL_MINUTES )); then
        echo "ERROR: active lock for ${city} (run=${existing_run}, age=${age_minutes}min)" >&2
        exit 1
      fi

      echo "WARN: stale lock for ${city} (run=${existing_run}, age=${age_minutes}min) — recovering" >&2
      exit_code=2
    else
      exit_code=0
    fi

    cat > "$lock" <<EOF
city: ${city}
run_id: ${run_id}
started_at: $(now_iso)
phase: ${phase}
ttl_minutes: ${TTL_MINUTES}
pid_hint: $$
EOF
    exit $exit_code
    ;;

  release)
    city="${1:?city required}"
    expected_run="${2:-}"

    lock="$(lock_path "$city")"
    if [[ ! -f "$lock" ]]; then
      echo "WARN: no lock to release for ${city}" >&2
      exit 0
    fi

    if [[ -n "$expected_run" ]]; then
      actual_run="$(grep -E '^run_id:' "$lock" | sed 's/^run_id: //; s/^"//; s/"$//')"
      if [[ "$actual_run" != "$expected_run" ]]; then
        echo "ERROR: lock run mismatch (expected=${expected_run}, actual=${actual_run}); not releasing" >&2
        exit 1
      fi
    fi

    rm -f "$lock"
    echo "OK released ${lock}"
    ;;

  update_phase)
    city="${1:?city required}"
    expected_run="${2:?run_id required}"
    new_phase="${3:?phase required}"

    lock="$(lock_path "$city")"
    if [[ ! -f "$lock" ]]; then
      echo "ERROR: no lock found for ${city}" >&2
      exit 1
    fi

    actual_run="$(grep -E '^run_id:' "$lock" | sed 's/^run_id: //; s/^"//; s/"$//')"
    if [[ "$actual_run" != "$expected_run" ]]; then
      echo "ERROR: lock run mismatch" >&2
      exit 1
    fi

    sed -i.bak "s|^phase:.*|phase: ${new_phase}|" "$lock"
    rm -f "${lock}.bak"
    echo "OK phase=${new_phase}"
    ;;

  inspect)
    city="${1:?city required}"
    lock="$(lock_path "$city")"
    if [[ -f "$lock" ]]; then
      cat "$lock"
    else
      echo "(no lock)"
    fi
    ;;

  *)
    echo "usage: $0 {acquire|release|update_phase|inspect} <city> [...]" >&2
    exit 2
    ;;
esac
