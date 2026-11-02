# Atlan integration

You can set up and manage your integration with [Atlan](https://atlan.com/) through the API. You need the following.

## Coalesce Quality client credentials

Go to Coalesce Quality ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) and create client credentials with the `Manage Extensions` and `Read Integration Settings` scopes. Set them in the environment as `QUALITY_CLIENT_ID` and `QUALITY_CLIENT_SECRET`.

## Atlan API token

Create an API token with the correct permissions in Atlan.

- In `Governance > Personas`, create a new persona. Give it a clear name, for example `Coalesce Quality API Access`.
    - In the persona, create these policies:
        - A `Domain Policy` with `Read` permission for domains. Choose `All Domains`, or only the domains you want to see in Coalesce Quality.
        - A `Metadata Policy` for each connection you want to see in Coalesce Quality. Choose the permission `Assets -> Read`.
- In `Admin > API Tokens`, create a new API token. Give it a clear name, for example `Coalesce Quality`.
    - Choose `Expiry Never`.
    - Add the persona you created in the step above (`Coalesce Quality API Access`).
    - Download or copy the API token.

Set these environment variables:
1. `ATLAN_TENANT_URL`: the URL you use to open Atlan (for example `https://<your-org>.atlan.com`).
2. `ATLAN_API_TOKEN`: the API token you created above.
