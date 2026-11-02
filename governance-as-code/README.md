# governance-as-code

Manage Coalesce Quality data products, owners, ownership and alert routing from
a YAML file, through the public API. Write the state you want once, `plan` it,
`apply` it, and run it again safely. You can also `export` your live workspace
into a file and `pull` later changes back into it. Your comments are kept.

The tool uses the public `synq.dataproducts.v2` and `synq.owners.v1` APIs. It
works the same for people and for agents (see [Use from agents](#use-from-agents)).

## Install

The generated API client is published on buf's Python registry, so the installer
needs that extra index.

With [uv](https://docs.astral.sh/uv/) (recommended), `pyproject.toml` already
declares the buf index:

```bash
# straight from the public repo:
uv tool install "git+https://github.com/getsynq/api.git#subdirectory=governance-as-code"
# or run once without installing:
uvx --from "git+https://github.com/getsynq/api.git#subdirectory=governance-as-code" governance-as-code status

# from a local checkout:
uv tool install .
```

With pipx:

```bash
pipx install . --pip-args="--extra-index-url https://buf.build/gen/python"
```

In a plain virtual environment (for development):

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python -m coalesce_governance status      # run as a module
```

The JSON Schema is published at
[`schemas.synq.io/governance-as-code/draft/governance.schema.json`](https://schemas.synq.io/governance-as-code/draft/governance.schema.json).
A copy ships with the tool at
[`coalesce_governance/governance.schema.json`](coalesce_governance/governance.schema.json).
Reference either one from the `$schema` header of your file (see [The definition file](#the-definition-file)).

## Authenticate

The YAML file never contains secrets. The tool looks for credentials in this
order:

1. A bearer token: `--token` or `QUALITY_TOKEN`.
2. OAuth2 client credentials: `--client-id` and `--client-secret`, or
   `QUALITY_CLIENT_ID` and `QUALITY_CLIENT_SECRET`. Create the client in
   Coalesce Quality under Settings → API
   ([EU](https://app.synq.io/settings/api), [US](https://app.us.synq.io/settings/api),
   [AU](https://app.au.synq.io/settings/api)) with the scopes listed under
   [Permissions](#permissions).
3. Browser sign-in: `governance-as-code login`. The token is cached in
   `$XDG_CONFIG_HOME/coalesce-quality` (`~/.config/coalesce-quality` when
   `XDG_CONFIG_HOME` is not set). Set `GOVERNANCE_HOME` to use another directory.

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
```

The `SYNQ_` forms of these variables (`SYNQ_TOKEN`, `SYNQ_CLIENT_ID`,
`SYNQ_CLIENT_SECRET`) also work. If both are set, the `QUALITY_` one wins.

## Permissions

The public `synq.dataproducts.v2` and `synq.owners.v1` services check scopes.
Grant the scopes for the commands you run:

| Commands | Scopes required |
|---|---|
| `status`, `plan`, `export`, `pull` (read only) | Read Data Products (`SCOPE_DATAPRODUCT_READ`), Read Owners (`SCOPE_OWNERS_READ`), Read Ownership (`SCOPE_OWNERSHIP_READ`) |
| `apply`, `prune` (read and write) | The three read scopes, plus Edit Data Products (`SCOPE_DATAPRODUCT_EDIT`), Edit Owners (`SCOPE_OWNERS_EDIT`), Edit Ownership (`SCOPE_OWNERSHIP_EDIT`) |

[`scopes.mdx`](../scopes.mdx) lists every scope and the methods it allows.

How you grant the scopes depends on the credential type:

- API client (client credentials): create it under Settings → API. Only a
  workspace Admin can create one. Developer and Analyst users can open the page,
  but cannot add a client. Give the client all six scopes above, or only the three read scopes
  for a read-only client.
- Browser sign-in (`login`): API access for users must be turned on for the
  workspace. If it is off, the consent screen shows "API access is turned off",
  and a workspace Admin can turn it on in Workspace settings. On the consent
  screen, select:
  - "Read-only access", which covers the three read scopes. This is enough for
    `status`, `plan`, `export` and `pull`.
  - "Manage governance", which covers the three edit scopes. `apply` and `prune`
    need it. Only Admin and Developer users see this option, because only those
    roles hold the edit scopes.

## Regions

Pick a region with `--region`, or with `QUALITY_REGION`. To use a different
host, set `--endpoint` or `QUALITY_API_ENDPOINT`.

| Region | `--region` | Endpoint |
|---|---|---|
| Europe (default) | `eu` | `developer.synq.io` |
| United States | `us` | `api.us.synq.io` |
| Australia | `au` | `api.au.synq.io` |

Flags win over environment variables, and an endpoint wins over a region.
`SYNQ_REGION` and `SYNQ_API_ENDPOINT` also work.

## Commands

```bash
governance-as-code                              # status: live workspace summary + next steps
governance-as-code plan   -f governance.yaml    # what would change (no writes)
governance-as-code apply  -f governance.yaml    # apply (asks for confirmation first)
governance-as-code apply  -f governance.yaml --yes   # apply without asking
governance-as-code prune  -f governance.yaml --yes   # delete what the file manages
governance-as-code export -o governance.yaml    # workspace -> YAML (explicit ids)
governance-as-code pull   -f governance.yaml    # refresh file fields from the API (keeps comments)
governance-as-code login  --region us           # browser sign-in
```

`-f` defaults to `governance.yaml`. Without `-o`, `export` writes to stdout.
In a development virtual environment, run `python -m coalesce_governance ...`
instead.

`prune` deletes the owners and data products in the file that exist in the
workspace. Deleting an owner also deletes its ownerships. The delete is
permanent, and the ids can then be used again.

## Shell completion

Tab completion uses [argcomplete](https://kislyuk.github.io/argcomplete/).
Print the setup for your shell and add it to your shell's rc file:

```bash
governance-as-code completion bash    # or: zsh | fish
# for example, add this to ~/.bashrc:
eval "$(register-python-argcomplete governance-as-code)"
```

## The definition file

The tool validates the file against
[`governance.schema.json`](coalesce_governance/governance.schema.json) before
any write. Reference the schema at the top of your file to get autocomplete and
inline validation in your editor:

```yaml
# yaml-language-server: $schema=https://schemas.synq.io/governance-as-code/draft/governance.schema.json
dataproducts:
  - title: "Sales & Orders"    # `id:` omitted -> adopted by title or generated, then written back
    folder: "Domains"
    priority: P1               # P1 | P2 | P3
    resolver_ql: 'and(with_type("table"), or(with_name("sales"), with_name("order")))'
owners:
  - title: "Sales Team"
    contacts:
      - slack: "#sales-alerts"
      - email: ["sales@example.com"]
      - users: ["alice@example.com"]     # workspace users (synq.users.v1.UsersService)
    ownerships:
      - dataproduct: "Sales & Orders"    # own a product by title or id, OR:
        # query: { name: "Critical", resolver_ql: 'with_type("model", filter=with_name("revenue"))' }
        alert:
          severities: [ERROR, FATAL]     # WARN | ERROR | FATAL (empty = no alerts)
          notify_upstream: true
          ongoing: disabled              # disabled | stream | { schedule: "0 9 * * MON" }
```

See [`governance.example.yaml`](governance.example.yaml) for a full example.

## Identity and idempotency

Identity works like the API: every resource has a UUID `id`. You can:

- set it yourself (`export` writes it for you), or
- leave it out. On the next run, the tool adopts an existing data product or
  owner with the same `title`, so you do not create a duplicate of something
  you already have. If there is no match, it generates a new id. Ownerships
  always get a new id. In all cases, the tool writes the id back into your file,
  so later runs are idempotent.

Because the ids are stable, `apply` updates resources instead of creating
duplicates, and `prune` deletes only what the file manages. Before any write,
the tool validates the file and checks that no `id` is used twice.

A typical round trip is `export -o f.yaml`, then edit, then `apply -f f.yaml`.
`pull -f f.yaml` refreshes the data products and owners in the file from the
workspace and keeps your comments.

## Writing selections (ResolverQL)

Data product membership and ownership queries use ResolverQL. A data product
definition is a leaf, so it cannot use `in_product` or `in_domain`. An ownership
query can reference data products and domains.

The server stores each query in a canonical form. `pull` replaces the
`resolver_ql` of each data product with that rendered form, so the next `plan`
shows no drift for it.

## Alerting

An owner's `contacts` set where alerts go. An ownership's `alert` sets which
severities send an alert and how repeat notifications work. Owned assets need no
separate alert rule. For delivery, the Slack or Microsoft Teams integration must
be connected, and the `users` emails must match users in the workspace.

## Use from agents

- With no arguments, the tool prints a live workspace summary and the next
  commands to run, instead of the full help text.
- Errors print as JSON on stdout. Exit codes: `0` ok, `1` error, `2` bad flags.
- In a non-interactive shell, `apply` and `prune` need `--yes`. Without it they
  stop with an error instead of waiting for a prompt.
- `plan --json` and `apply --json` print the full list of actions with a
  `total`.
- Writes are idempotent, so a retry is safe.
