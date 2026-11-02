# Users management

This example shows how to list the members of a Coalesce Quality workspace and
manage their roles using the `UsersService` API.

`users.py` is a small command-line tool:

- `list` shows every member with their email, name, roles, invitation status
  and last interactive sign-in.
- `add-role` and `remove-role` grant or withdraw one role.
- `set-roles` gives a user exactly the given roles. It adds the missing roles
  before it removes the extra ones.

## Prerequisites

Create client credentials at
Coalesce Quality → API settings ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) with the
Read Users scope (for `list`) and the Edit Users scope (for the role
commands), and export them:

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# Optional. The default is the EU endpoint, developer.synq.io.
# Other regions: api.us.synq.io for US, api.au.synq.io for AU.
export QUALITY_API_ENDPOINT=developer.synq.io
```

The variables can also be placed in a `.env` file next to the script.

## Setup and running

```bash
pip install -r requirements.txt

python users.py list
python users.py add-role jane@example.com analyst
python users.py remove-role jane@example.com analyst
python users.py set-roles jane@example.com developer analyst
```

## Naming a user

The role commands take a user identity: any of the strings a user has in
`User.identities`: `synq:<user_id>`, `email:<email>`, `slack:<slack_user_id>`
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
- Granting a role to someone who is not a member yet makes them a member. You
  can name such a user only by their `synq:` or `email:` identity.
- Withdrawing a user's last role removes them from the workspace, together
  with their linked aliases. `set-roles` adds before it removes, so changing a
  user's role never removes them in between.
- `last_login_at` records interactive sign-ins only. Token refreshes and API
  calls do not count. It prints as `never` when no sign-in has been recorded.
