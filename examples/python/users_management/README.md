# Users Management

This example shows how to list the members of a Coalesce Quality workspace and
manage their roles using the `UsersService` API.

`users.py` is a small command-line tool:

- **`list`** — every member with their email, name, roles, invitation status and
  last interactive sign-in.
- **`add-role`** / **`remove-role`** — grant or withdraw a single role.
- **`set-roles`** — make a user hold exactly the given roles, adding the missing
  ones before removing the extra ones.

## Prerequisites

Create client credentials at
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

The variables can also be placed in a `.env` file next to the script.

## Setup & running

```bash
pip install -r requirements.txt

python users.py list
python users.py add-role jane@example.com analyst
python users.py remove-role jane@example.com analyst
python users.py set-roles jane@example.com developer analyst
```

## Naming a user

The role commands take a user **identity**: any of the strings a user carries in
`User.identities` — `synq:<user_id>`, `email:<email>`, `slack:<slack_user_id>`
or `msteams:<member_id>`. The script accepts a bare email address as shorthand
for `email:<address>`.

## Roles

| Argument        | API value                 | Shown in Settings → Team |
| --------------- | ------------------------- | ------------------------ |
| `admin`         | `USER_ROLE_ADMIN`         | Admin                    |
| `developer`     | `USER_ROLE_DEVELOPER`     | Developer                |
| `analyst`       | `USER_ROLE_ANALYST`       | Analyst                  |
| `business_user` | `USER_ROLE_BUSINESS_USER` | Business User            |

## Behaviour to be aware of

- `AddUserRole` and `RemoveUserRole` are idempotent: granting a role the user
  already holds, or withdrawing one they do not hold, succeeds without change.
- Granting a role to someone who is not yet a member makes them a member. Such a
  user can only be named by their `synq:` or `email:` identity.
- **Withdrawing a user's last role removes them from the workspace**, together
  with their linked aliases. `set-roles` adds before it removes, so changing a
  user's role never removes them in between.
- `last_login_at` records interactive sign-ins only; token refreshes and API
  calls do not count. It prints as `never` when no sign-in has been recorded.
