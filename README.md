<div align="center">

# ⚡ VESC Express for Home Assistant

**Battery and temperature readings from your VESC motor controller – right in Home Assistant.**

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Architectures](https://img.shields.io/badge/arch-aarch64%20%7C%20amd64%20%7C%20armv7-informational)
![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white)
![MQTT Discovery](https://img.shields.io/badge/MQTT-Discovery-660066?logo=mqtt&logoColor=white)

[![Add repository to Home Assistant](https://my.home-assistant.io/badges/supervisor_add_addon_repository.svg)](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FJannik44%2Fha-vesc)

</div>

---

## 📊 Sensors

| | Sensor | Entity | Unit |
|:-:|---|---|:-:|
| 🔋 | Battery voltage | `sensor.vesc_voltage` | V |
| 📈 | State of charge | `sensor.vesc_soc` | % |
| 🌡️ | ESC temperature | `sensor.vesc_temp_esc` | °C |
| ⚙️ | Motor temperature | `sensor.vesc_temp_motor` | °C |

## ✨ Features

- **No extra server** – replaces a standalone bridge and runs directly as an add-on
- **Automatic CAN detection** – finds the motor controller behind the VESC Express via `COMM_PING_CAN`
- **MQTT discovery** – sensors show up as a **VESC** device, renamable and with long-term statistics
- **Fallback without MQTT** – writes the values through the Home Assistant REST API
- **Small & fast** – a single static Go binary, no runtime dependencies

## 🚀 Installation

1. Click the button above **or** add it manually:
   *Settings → Add-ons → Add-on Store → ⋮ → Repositories* →
   enter `https://github.com/Jannik44/ha-vesc`
2. Select **VESC Express** in the store and install it
   *(the add-on is built on your HA device the first time – this takes a few minutes)*
3. Check the options and hit **Start**

> 💡 For the best integration, set up the **Mosquitto broker** add-on and the **MQTT** integration – the add-on uses them automatically.

## ⚙️ Configuration

```yaml
host: 192.168.0.119   # IP or hostname of the VESC Express
port: 65102           # TCP port of the VESC Express
can_id: -1            # -1 = auto-detect
cell_count: 20        # cells in series (S), used for the state of charge
scan_interval: 60     # polling interval in seconds
# optional, only for an external MQTT broker:
# mqtt_host: 192.168.0.10
# mqtt_port: 1883
# mqtt_user: user
# mqtt_password: secret
```

## 🔧 How it works

```
┌──────────────┐  TCP :65102   ┌──────────────┐   CAN   ┌────────────────┐
│  HA add-on   │ ────────────► │ VESC Express │ ──────► │motor controller│
│ vesc_express │ ◄──────────── │              │ ◄────── │     (VESC)     │
└──────┬───────┘ GET_VALUES    └──────────────┘         └────────────────┘
       │ MQTT discovery / REST API
       ▼
┌──────────────┐
│Home Assistant│
└──────────────┘
```

The **state of charge** is derived from the battery voltage using a typical Li-ion discharge curve
(4.20 V/cell = 100 %, 3.70 V = 50 %, 2.90 V = 0 %, linearly interpolated).

If the VESC can't be reached (e.g. the vehicle is switched off), the last values are kept – the error shows up in the add-on log.

## 🩺 Troubleshooting

| Log message | Cause |
|---|---|
| `connect: connection refused` / `i/o timeout` | VESC Express not reachable – check IP/port |
| `COMM_PING_CAN Timeout` | Express reachable, but no controller on the CAN bus (switched off?) |
| `keine GET_VALUES Antwort` | Wrong `can_id` – set it to `-1` |
| `Kein MQTT verfügbar … REST-API` | No Mosquitto found, REST fallback active |

## 🛠️ Development

Read the values once locally (without Home Assistant):

```bash
cd vesc_express/app && go run . -once -options /dev/null
```

More details: [vesc_express/DOCS.md](vesc_express/DOCS.md)
