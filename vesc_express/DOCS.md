# VESC Express

Fragt den Motorcontroller über den VESC Express (TCP, Port 65102, `COMM_FORWARD_CAN` → `COMM_GET_VALUES`) ab und stellt in Home Assistant bereit:

| Entität | Einheit |
|---|---|
| `sensor.vesc_voltage` – Akkuspannung | V |
| `sensor.vesc_soc` – Ladestand | % |
| `sensor.vesc_temp_esc` – ESC Temperatur | °C |
| `sensor.vesc_temp_motor` – Motor Temperatur | °C |

## Wie die Werte nach Home Assistant kommen

- **MQTT (empfohlen):** Ist das Mosquitto-Add-on mit der MQTT-Integration eingerichtet, wird es automatisch genutzt. Die Sensoren erscheinen per MQTT-Discovery als Gerät **VESC** mit eindeutigen IDs (umbenennbar, Statistik/Langzeitdaten funktionieren). Werte werden „retained“ gesendet und überstehen einen HA-Neustart.
- **Ohne MQTT:** Fallback über die Home-Assistant-REST-API. Die Sensoren funktionieren, haben aber keine eindeutige ID und sind bis zur nächsten Abfrage nach einem HA-Neustart unbekannt.

Ist der VESC nicht erreichbar (z. B. Fahrzeug aus), bleiben die letzten Werte stehen; der Fehler steht im Add-on-Log.

## Optionen

- `host` / `port` – Adresse des VESC Express
- `can_id` – `-1` ermittelt die CAN-ID automatisch
- `cell_count` – Zellen in Reihe (Standard 20S). Der Ladestand wird aus der Spannung über eine typische Li-Ion-Kurve (4,20 V = 100 %, 2,90 V = 0 %) berechnet.
- `scan_interval` – Abfrageintervall in Sekunden
- `mqtt_*` – nur nötig für einen externen Broker
