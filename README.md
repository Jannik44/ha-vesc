# VESC Add-ons für Home Assistant

## Installation

1. Einstellungen → Add-ons → Add-on Store → ⋮ → **Repositories**
2. URL dieses Repositories eintragen: `https://github.com/Jannik44/ha-vesc`
3. Add-on **VESC Express** installieren, Optionen prüfen, starten.

Details: [vesc_express/DOCS.md](vesc_express/DOCS.md)

## Lokal testen

```bash
cd vesc_express/app && go run . -once -options /dev/null
```
