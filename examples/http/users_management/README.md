# Users Management (HTTP/JSON)

This example shows how to list the members of a Coalesce Quality workspace and
manage their roles by calling the `UsersService` API over plain HTTP/JSON, for
clients that cannot use gRPC. It mirrors the
[Python users management example](../../python/users_management).

`users.sh` is a small command-line tool built on `curl` and `jq`:

- **`list`** — every member with their email, name, roles, invitation status and
  last interactive sign-in.
- **`add-role`** / **`remove-role`** — grant or withdraw a single role.
- **`set-roles`** — make a user hold exactly the given roles, adding the missing
  ones before removing the extra ones.

## Prerequisites

Install `curl` and `jq`. Create client credentials at
[Coalesce Quality → API settings](https://app.synq.io/settings/api) with the
**Read Users** scope (for `list`) and the **Edit Users** scope (for the role
commands), and export them:

```bash
export SYNQ_CLIENT_ID=...
export SYNQ_CLIENT_SECRET=...
# Optional; defaults to the EU endpoint developer.synq.io.
# For other regions: api.us.synq.io for US, api.au.synq.io for AU.
export API_ENDPOINT=developer.synq.io
```

The variables can also be placed in a `.env` file in the directory you run the
script from.

## Running

```bash
./users.sh list
./users.sh add-role jane@example.com analyst
./users.sh remove-role jane@example.com analyst
./users.sh set-roles jane@example.com developer analyst
```

## Endpoints

Obtain an access token with the OAuth2 client credentials grant, then send it as
`Authorization: Bearer <access_token>` on every call. Request and response
bodies are JSON with `Content-Type: application/json`.

```bash
curl -s https://developer.synq.io/oauth2/token \
  -d client_id="$SYNQ_CLIENT_ID" \
  -d client_secret="$SYNQ_CLIENT_SECRET" \
  -d grant_type=client_credentials
```

| Method | Path                            | RPC                 | Request                              |
| ------ | ------------------------------- | ------------------- | ------------------------------------ |
| GET    | `/api/users/v1`                 | `ListUsers`         | —                                    |
| POST   | `/api/users/v1/batch-get`       | `BatchGetUsers`     | `{"userIds": [...]}`                 |
| GET    | `/api/users/v1/by-email`        | `GetByEmail`        | `?email=<address>`                   |
| POST   | `/api/users/v1/by-alias`        | `GetByAlias`        | `{"alias": {...}}`                   |
| GET    | `/api/users/v1/identities`      | `GetUserIdentities` | `?user_id=<id>` (omit for yourself)  |
| GET    | `/api/users/v1/aliases`         | `GetUserAliases`    | `?user_id=<id>`                      |
| PUT    | `/api/users/v1/aliases`         | `UpsertUserAliases` | `{"aliases": [...]}`                 |
| POST   | `/api/users/v1/aliases/delete`  | `DeleteUserAliases` | `{"aliases": [...]}`                 |
| POST   | `/api/users/v1/roles`           | `AddUserRole`       | `{"identity": "...", "role": "..."}` |
| POST   | `/api/users/v1/roles/remove`    | `RemoveUserRole`    | `{"identity": "...", "role": "..."}` |

JSON field names are lowerCamelCase (`firstName`, `lastLoginAt`), enums are
sent and returned by name (`USER_ROLE_ANALYST`, `USER_STATUS_ACTIVE`) and
timestamps are RFC 3339 strings. Fields left at their default value — an empty
list, an unset timestamp — are omitted from responses.

## Connecting an identity-governance tool

Access-review and identity-governance products with a generic web-services or
REST connector can aggregate workspace members and their roles from these
endpoints:

- **Authentication** — OAuth2 client credentials against
  `https://<endpoint>/oauth2/token`, with the credentials above.
- **Account aggregation** — `GET /api/users/v1`, accounts at the JSON path
  `$.users`. The response is not paginated. Use `id` (or `email`) as the account
  identifier; `email`, `firstName`, `lastName`, `status` and `lastLoginAt` as
  account attributes; and `roles` as the multi-valued entitlement.
- **Entitlement aggregation** — the four roles in the table below are the
  complete set.
- **Provisioning** — `POST /api/users/v1/roles` to grant an entitlement and
  `POST /api/users/v1/roles/remove` to revoke one, naming the account by
  `email:<email>` or `synq:<id>`.

## Naming a user

The role commands take a user **identity**: any of the strings a user carries in
`identities` — `synq:<user_id>`, `email:<email>`, `slack:<slack_user_id>` or
`msteams:<member_id>`. The script accepts a bare email address as shorthand for
`email:<address>`.

## Roles

| Argument        | API value                 | Shown in Settings → Team |
| --------------- | ------------------------- | ------------------------ |
| `admin`         | `USER_ROLE_ADMIN`         | Admin                    |
| `developer`     | `USER_ROLE_DEVELOPER`     | Developer                |
| `analyst`       | `USER_ROLE_ANALYST`       | Analyst                  |
| `business_user` | `USER_ROLE_BUSINESS_USER` | Business User            |

## Behaviour to be aware of

- Granting and withdrawing roles is idempotent: granting a role the user already
  holds, or withdrawing one they do not hold, succeeds without change.
- Granting a role to someone who is not yet a member makes them a member. Such a
  user can only be named by their `synq:` or `email:` identity.
- **Withdrawing a user's last role removes them from the workspace**, together
  with their linked aliases, and the response then carries no `user`.
  `set-roles` adds before it removes, so changing a user's role never removes
  them in between.
- `lastLoginAt` records interactive sign-ins only; token refreshes and API calls
  do not count. It is absent, and prints as `never`, when no sign-in has been
  recorded.
