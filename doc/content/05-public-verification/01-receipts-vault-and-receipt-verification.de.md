---
title-en: Receipts Vault and Receipt Verification
title-de: Receipt Vault und Belegprüfung
---

# Receipt Vault und Belegprüfung

![Receipt Vault](../../assets/captures/app-receipts-vault.en.light.desktop.png)

## Nutzerseite

- Receipt-Vault-Oberfläche: `GET /receipts`

## Verifikations-APIs

- Offenen Beleg prüfen: `POST /api/votes/verify/open`
- Geheimen Beleg prüfen: `POST /api/votes/verify/secret`

## Erwartete Eingaben

- `vote_id` und `receipt_token` sind erforderlich.
- Bei offenen Abstimmungen kann optional `attendee_id` gesendet werden.
- Ungültige Daten oder unbekannte Belege liefern Fehlerantworten.
