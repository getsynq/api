#!/usr/bin/env bash
#
# Users Management Example (HTTP/JSON) - list workspace members and manage their roles
#
# Commands:
#     ./users.sh list
#     ./users.sh add-role <identity> <role>
#     ./users.sh remove-role <identity> <role>
#     ./users.sh set-roles <identity> <role> [<role> ...]
#
# <identity> is an email address or any identity string from `list`
# ("synq:<user_id>", "email:<email>", "slack:<id>", "msteams:<id>").
# <role> is one of: admin, developer, analyst, business_user.
#
# Prerequisites:
# - curl and jq
# - SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET environment variables
#   (Read Users scope for `list`, Edit Users scope for the role commands)
# - Optional API_ENDPOINT (defaults to developer.synq.io)

set -euo pipefail

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

API_ENDPOINT="${API_ENDPOINT:-developer.synq.io}"
BASE_URL="https://${API_ENDPOINT}/api/users/v1"

if [[ -z "${SYNQ_CLIENT_ID:-}" || -z "${SYNQ_CLIENT_SECRET:-}" ]]; then
  echo "SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set" >&2
  exit 1
fi

ROLE_NAMES=(admin developer analyst business_user)

usage() {
  sed -n '3,13p' "$0" | sed 's/^# \{0,1\}//' >&2
  exit 2
}

role_value() {
  case "$1" in
    admin) echo USER_ROLE_ADMIN ;;
    developer) echo USER_ROLE_DEVELOPER ;;
    analyst) echo USER_ROLE_ANALYST ;;
    business_user) echo USER_ROLE_BUSINESS_USER ;;
    *)
      echo "unknown role '$1', expected one of: ${ROLE_NAMES[*]}" >&2
      exit 2
      ;;
  esac
}

# Accepts a bare email address as shorthand for "email:<address>".
to_identity() {
  if [[ "$1" != *:* && "$1" == *@* ]]; then
    echo "email:$1"
  else
    echo "$1"
  fi
}

TOKEN=""

obtain_token() {
  TOKEN=$(curl -sS --fail-with-body "https://${API_ENDPOINT}/oauth2/token" \
    --data-urlencode "client_id=${SYNQ_CLIENT_ID}" \
    --data-urlencode "client_secret=${SYNQ_CLIENT_SECRET}" \
    --data-urlencode "grant_type=client_credentials" | jq -r '.access_token')
}

api() {
  local method="$1" path="$2" body="${3:-}"
  local args=(-sS --fail-with-body -X "$method" "${BASE_URL}${path}" -H "Authorization: Bearer ${TOKEN}")
  if [[ -n "$body" ]]; then
    args+=(-H "Content-Type: application/json" --data "$body")
  fi
  curl "${args[@]}"
}

# Fields left at their default are omitted from the JSON, hence the `//` fallbacks.
FORMAT_USER='
  def role: {USER_ROLE_ADMIN: "admin", USER_ROLE_DEVELOPER: "developer",
             USER_ROLE_ANALYST: "analyst", USER_ROLE_BUSINESS_USER: "business_user"}[.] // .;
  def status: {USER_STATUS_ACTIVE: "active", USER_STATUS_INVITED: "invited"}[. // ""] // "-";
  [ (.email // ""),
    ((((.firstName // "") + " " + (.lastName // "")) | ltrimstr(" ") | rtrimstr(" ")) | if . == "" then "-" else . end),
    ((.roles // []) | map(role) | join(",") | if . == "" then "-" else . end),
    (.status | status),
    (.lastLoginAt // "never")
  ] | @tsv'

print_users() {
  jq -r "$FORMAT_USER" | awk -F'\t' '{ printf "%-40s %-30s %-30s %-8s %s\n", $1, $2, $3, $4, $5 }'
}

list_users() {
  local resp
  resp=$(api GET "")
  printf "%-40s %-30s %-30s %-8s %s\n" EMAIL NAME ROLES STATUS "LAST LOGIN"
  jq -c '(.users // []) | sort_by(.email) | .[]' <<<"$resp" | print_users
  printf "\n%s users\n" "$(jq '(.users // []) | length' <<<"$resp")"
}

add_role() {
  local identity="$1" role="$2" resp
  resp=$(api POST /roles "$(jq -n --arg identity "$identity" --arg role "$(role_value "$role")" '{identity: $identity, role: $role}')")
  echo "Added ${role} to ${identity}"
  jq -c '.user' <<<"$resp" | print_users
}

remove_role() {
  local identity="$1" role="$2" resp
  resp=$(api POST /roles/remove "$(jq -n --arg identity "$identity" --arg role "$(role_value "$role")" '{identity: $identity, role: $role}')")
  echo "Removed ${role} from ${identity}"
  if jq -e '.user' <<<"$resp" >/dev/null; then
    jq -c '.user' <<<"$resp" | print_users
  else
    echo "User is not a member of the workspace"
  fi
}

current_roles() {
  local identity="$1" email=""
  if [[ "$identity" == email:* ]]; then
    email=$(tr '[:upper:]' '[:lower:]' <<<"${identity#email:}")
  fi
  api GET "" | jq -r --arg identity "$identity" --arg email "$email" '
    def role: {USER_ROLE_ADMIN: "admin", USER_ROLE_DEVELOPER: "developer",
               USER_ROLE_ANALYST: "analyst", USER_ROLE_BUSINESS_USER: "business_user"}[.] // empty;
    [ (.users // [])[]
      | select(((.identities // []) | index($identity)) or ($email != "" and ((.email // "") | ascii_downcase) == $email)) ]
    | first // {} | (.roles // [])[] | role'
}

# Adds before removing so the user never loses membership midway.
set_roles() {
  local identity="$1"
  shift
  local wanted current role
  for role in "$@"; do role_value "$role" >/dev/null; done
  wanted=$(printf '%s\n' "$@" | sort -u)
  current=$(current_roles "$identity" | sort -u)

  if [[ "$wanted" == "$current" ]]; then
    echo "${identity} already holds $(paste -sd, - <<<"$wanted" | sed 's/,/, /g')"
    return
  fi
  for role in $(comm -23 <(echo "$wanted") <(echo "$current")); do
    add_role "$identity" "$role"
  done
  for role in $(comm -13 <(echo "$wanted") <(echo "$current")); do
    remove_role "$identity" "$role"
  done
}

main() {
  [[ $# -ge 1 ]] || usage
  local command="$1"
  shift

  case "$command" in
    list)
      [[ $# -eq 0 ]] || usage
      obtain_token
      list_users
      ;;
    add-role | remove-role)
      [[ $# -eq 2 ]] || usage
      role_value "$2" >/dev/null
      obtain_token
      "${command/-/_}" "$(to_identity "$1")" "$2"
      ;;
    set-roles)
      [[ $# -ge 2 ]] || usage
      local identity
      identity=$(to_identity "$1")
      shift
      obtain_token
      set_roles "$identity" "$@"
      ;;
    *)
      usage
      ;;
  esac
}

main "$@"
