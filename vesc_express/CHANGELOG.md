# Changelog

## 1.0.1
- Fix: SUPERVISOR_TOKEN fehlte (s6-Init hat die Umgebung geleert) – Add-on startet das Programm jetzt direkt
- Kein Absturz-Loop mehr, wenn Home Assistant nicht erreichbar ist (neuer Versuch alle 30 s)

## 1.0.0
- Erste Version: Akkuspannung, Ladestand, ESC- und Motortemperatur per MQTT-Discovery (Fallback REST-API)
