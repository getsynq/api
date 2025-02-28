# Atlan Integration

You can set up and manage your integration with [Atlan](https://atlan.com/) using the API. To set it up, you will need the following.

## SYNQ Client Credentials

Head over to [SYNQ](https://app.synq.io/settings/api) to create client credentials with the `Manage Extensions` scope. Set the credentials in the environment as `SYNQ_CLIENT_ID` and `SYNQ_CLIENT_SECRET`.

## Atlan API token

You will need to create an API token with appropriate permissions on Atlan.

- In `Governance > Personas` , create a new persona. Name it something indicative like `SYNQ API Access`.
    - Within the persona, create the following policies:
        - `Domain Policy` with `Read` permission for domains. You can choose `All Domains` or cherry pick the ones you want to be visible on SYNQ.
        - `Metadata Policy` for each connection that you want to be visible on SYNQ. Choose the permission `Assets -> Read`
- In `Admin > API Tokens` create a new API token. Name it something indicative like `SYNQ`
    - Choose `Expiry Never`
    - Add the persona you just created in the step above (`SYNQ API Access`)
    - Download or copy the API token that you see.

Set the following enviroment variables:
1. `ATLAN_TENANT_URL` - This is the URL you use to access Atlan (eg. `https://<your-org>.atlan.com`)
2. `ATLAN_API_TOKEN`  - This is the API token you generated above.
