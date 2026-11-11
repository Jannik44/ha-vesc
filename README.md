<div align="center">

# ⚡ VESC Express für Home Assistant

**Akku- und Temperaturwerte deines VESC-Motorcontrollers – direkt in Home Assistant.**

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Architekturen](https://img.shields.io/badge/arch-aarch64%20%7C%20amd64%20%7C%20armv7-informational)
![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)
![MQTT Discovery](https://img.shields.io/badge/MQTT-Discovery-660066?logo=mqtt&logoColor=white)

[![Repository zu Home Assistant hinzufügen](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FJannik44%2Fha-vesc)

</div>

---

## 📊 Sensoren

| | Sensor | Entität | Einheit |
|:-:|---|---|:-:|
| 🔋 | Akkuspannung | `sensor.vesc_voltage` | V |
| 📈 | Ladestand | `sensor.vesc_soc` | % |
| 🌡️ | ESC Temperatur | `sensor.vesc_temp_esc` | °C |
| ⚙️ | Motor Temperatur | `sensor.vesc_temp_motor` | °C |

## ✨ Features

- **Kein Extra-Server** – ersetzt eine separate Bridge, läuft direkt als Add-on
- **Automatische CAN-Erkennung** – findet den Motorcontroller hinter dem VESC Express per `COMM_PING_CAN`
- **MQTT-Discovery** – Sensoren erscheinen als Gerät **VESC**, umbenennbar und mit Langzeitstatistik
- **Fallback ohne MQTT** – schreibt die Werte über die Home-Assistant-REST-API
- **Klein & schnell** – ein einzelnes statisches Go-Binary, keine Laufzeitabhängigkeiten

## 🚀 Installation

1. Auf den Button oben klicken **oder** manuell:
   *Einstellungen → Add-ons → Add-on Store → ⋮ → Repositories* →
   `https://github.com/Jannik44/ha-vesc` eintragen
2. **VESC Express** im Store auswählen und installieren
   *(das Add-on wird beim ersten Mal auf deinem HA-Gerät gebaut – das dauert ein paar Minuten)*
3. Optionen prüfen und **Starten**

> 💡 Für die beste Integration das **Mosquitto-Broker**-Add-on und die **MQTT**-Integration einrichten – das Add-on nutzt sie automatisch.

## ⚙️ Konfiguration

```yaml
host: 192.168.0.119   # IP oder Hostname des VESC Express
port: 65102           # TCP-Port des VESC Express
can_id: -1            # -1 = automatisch erkennen
cell_count: 20        # Zellen in Reihe (S) für die Ladestand-Berechnung
scan_interval: 60     # Abfrageintervall in Sekunden
# optional, nur für einen externen MQTT-Broker:
# mqtt_host: 192.168.0.10
# mqtt_port: 1883
# mqtt_user: user
# mqtt_password: geheim
```

## 🔧 So funktioniert's

```
┌──────────────┐  TCP :65102   ┌──────────────┐   CAN   ┌────────────────┐
│  HA Add-on   │ ────────────► │ VESC Express │ ──────► │ Motorcontroller│
│ (vesc_express│ ◄──────────── │              │ ◄────── │     (VESC)     │
└──────┬───────┘ GET_VALUES    └──────────────┘         └────────────────┘
       │ MQTT-Discovery / REST-API
       ▼
┌──────────────┐
│Home Assistant│
└──────────────┘
```

Der **Ladestand** wird aus der Akkuspannung über eine typische Li-Ion-Entladekurve berechnet
(4,20 V/Zelle = 100 %, 3,70 V = 50 %, 2,90 V = 0 %, linear interpoliert).

Ist der VESC nicht erreichbar (z. B. Fahrzeug ausgeschaltet), bleiben die letzten Werte erhalten – der Fehler steht im Add-on-Log.

## 🩺 Fehlersuche

| Log-Meldung | Ursache |
|---|---|
| `connect: connection refused` / `i/o timeout` | VESC Express nicht erreichbar – IP/Port prüfen |
| `COMM_PING_CAN Timeout` | Express erreichbar, aber kein Controller am CAN-Bus (ausgeschaltet?) |
| `keine GET_VALUES Antwort` | Falsche `can_id` – auf `-1` setzen |
| `Kein MQTT verfügbar … REST-API` | Kein Mosquitto gefunden, Fallback aktiv |

## 🛠️ Entwicklung

Lokal einmal auslesen (ohne Home Assistant):

```bash
cd vesc_express/app && go run . -once -options /dev/null
```

Weitere Details: [vesc_express/DOCS.md](vesc_express/DOCS.md)
