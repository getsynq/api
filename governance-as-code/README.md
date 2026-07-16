# governance-as-code

Manage **Coalesce Quality** data products, owners, ownership and alert routing
declaratively from a YAML file, via the public API. Write your desired state
once, `plan` it, `apply` it, re-run it safely — or `export` your live workspace
into a file and `pull` updates back while keeping your comments.

Built on the public `synq.dataproducts.v2` and `synq.owners.v1` APIs. Designed to
be pleasant for both humans and agents (see *Agent-friendly*).

## Install

The generated API client lives on buf's Python registry, so point the installer
at that extra index.

**With [uv](https://docs.astral.sh/uv/) (recommended)** — `pyproject.toml`
declares the buf index, so this just works:

```bash
# straight from the public repo:
uv tool install "git+https://github.com/getsynq/api.git#subdirectory=governance-as-code"
# or run once without installing:
uvx --from "git+https://github.com/getsynq/api.git#subdirectory=governance-as-code" governance-as-code status

# from a local checkout:
uv tool install .
```

**With pipx:**

```bash
pipx install . --pip-args="--extra-index-url https://buf.build/gen/python"
```

**Plain venv (dev):**

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt
python -m coalesce_governance status      # run as a module
```

The schema is published at
[`schemas.synq.io/governance-as-code/draft/governance.schema.json`](https://schemas.synq.io/governance-as-code/draft/governance.schema.json).

## Authenticate (kept separate from your definition file)

The YAML never contains secrets. Provide credentials one of three ways — the
tool tries them in this order:

1. **Bearer token** — `--token` or `QUALITY_TOKEN`
2. **OAuth2 client credentials** — `--client-id/--client-secret` or
   `QUALITY_CLIENT_ID` + `QUALITY_CLIENT_SECRET`
   (create at [Coalesce Quality → API settings](https://app.synq.io/settings/api)
   with scopes: Read/Edit Data Products, Owners, Ownership)
3. **Browser sign-in** — `governance-as-code login` (caches a token under
   `$XDG_CONFIG_HOME/coalesce-quality`, override with `GOVERNANCE_HOME`)

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
```

## Regions

Pick a region with `--region` (or set `--endpoint` / `API_ENDPOINT` directly):

| Region | `--region` | Endpoint |
|---|---|---|
| Europe (default) | `eu` | `developer.synq.io` |
| United States | `us` | `api.us.synq.io` |
| APAC | `apac` | *coming soon* |

## Commands

```bash
governance-as-code                              # status: live workspace summary + next steps
governance-as-code plan   -f governance.yaml    # what would change (no writes)
governance-as-code apply  -f governance.yaml     # apply (confirms first)
governance-as-code apply  -f governance.yaml --yes   # apply without prompting
governance-as-code prune  -f governance.yaml --yes   # delete what the file manages
governance-as-code export -o governance.yaml     # workspace -> YAML (explicit ids)
governance-as-code pull   -f governance.yaml     # refresh file fields from API (keeps comments)
governance-as-code login  --region us            # browser sign-in
```

(In a dev venv without installing, use `python -m coalesce_governance ...`.)

## Shell completion

Tab-completion is powered by [argcomplete](https://kislyuk.github.io/argcomplete/).
Print the setup for your shell and add it to your rc file:

```bash
governance-as-code completion bash    # or: zsh | fish
# e.g. add to ~/.bashrc:
eval "$(register-python-argcomplete governance-as-code)"
```

## The definition file

Validated against [`governance.schema.json`](governance.schema.json) before any
write. Reference the schema at the top of your file for editor autocomplete and
inline validation:

```yaml
# yaml-language-server: $schema=./governance.schema.json
dataproducts:
  - title: "Sales & Orders"    # `id:` omitted -> adopted-by-title or generated, then written back
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

See [`governance.example.yaml`](governance.example.yaml).

## Identity & idempotency

Identity works like the API: every resource has a UUID **`id`**. You can:

- **supply it** (as `export` emits), or
- **omit it** — on the next run the tool either **adopts** an existing resource
  with the same `title` (so you don't create duplicates of things you already
  have), or **generates** a new one. Either way it **writes the id back** into
  your file so subsequent runs are idempotent.

Because ids are stable, `apply` converges instead of duplicating and `prune`
deletes exactly what the file manages. The file is validated before any write,
including a check that no `id` is used twice. Round-trip:
`export -o f.yaml` → edit → `apply -f f.yaml`; `pull -f f.yaml` refreshes fields
from the workspace while keeping your comments.

## Authoring selections (ResolverQL)

Membership (products) and ownership selections use ResolverQL. Data-product
definitions are **leaves** (no `in_dataproduct` / `in_domain`); ownership queries
may reference products and domains. The server stores queries canonically and
`pull` rewrites your `resolver_ql` to the rendered form (so a subsequent `plan`
shows no drift).

## Alerting

An owner's `contacts` are *where* alerts go; an ownership's `alert` (severities +
ongoing strategy) is *what/when* fires — no separate alert rule for owned assets.
For delivery, the Slack/Teams integration must be connected and `users` emails
must match workspace users.

## Agent-friendly

- **Content-first:** no arguments prints a live workspace summary + next-step
  command templates, not a wall of help.
- **Structured errors & exit codes:** errors print as JSON on stdout; `0` ok,
  `1` error, `2` bad flags.
- **No hanging prompts:** in a non-interactive shell `apply`/`prune` require
  `--yes` (they fail fast telling you so, rather than blocking on a prompt).
- **Machine-readable plans:** `plan --json` / `apply --json` emit the full action
  list with a `total`.
- **Idempotent mutations:** safe to retry.
