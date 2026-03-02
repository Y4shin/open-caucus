---
title-en: Receipts Vault and Receipt Verification
title-de: Receipt Vault und Belegprüfung
---

# Receipts Vault and Receipt Verification

![Receipts vault](../../assets/captures/app-receipts-vault.en.light.desktop.png)

## User page

- Receipt vault UI: `GET /receipts`

## Verification APIs

- Verify open receipt: `POST /api/votes/verify/open`
- Verify secret receipt: `POST /api/votes/verify/secret`

## Input expectations

- `vote_id` and `receipt_token` are required.
- Open vote verification can include `attendee_id` when available.
- Invalid payloads or unknown receipts return an error response.
