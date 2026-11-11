package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"
)

type Options struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	CanID        int    `json:"can_id"`
	CellCount    int    `json:"cell_count"`
	ScanInterval int    `json:"scan_interval"`
	MQTTHost     string `json:"mqtt_host"`
	MQTTPort     int    `json:"mqtt_port"`
	MQTTUser     string `json:"mqtt_user"`
	MQTTPassword string `json:"mqtt_password"`
}

// Li-Ion Entladekurve: [Spannung pro Zelle, SOC %]
var socCurve = [][2]float64{
	{4.20, 100}, {4.10, 90}, {4.00, 80}, {3.90, 70},
	{3.80, 60}, {3.70, 50}, {3.60, 40}, {3.50, 30},
	{3.40, 20}, {3.30, 10}, {3.20, 5}, {2.90, 0},
}

func calcSOC(packV float64, cells int) float64 {
	cellV := packV / float64(cells)
	if cellV >= socCurve[0][0] {
		return 100
	}
	last := socCurve[len(socCurve)-1]
	if cellV <= last[0] {
		return 0
	}
	for i := 0; i < len(socCurve)-1; i++ {
		v1, s1 := socCurve[i][0], socCurve[i][1]
		v2, s2 := socCurve[i+1][0], socCurve[i+1][1]
		if cellV <= v1 && cellV >= v2 {
			return round1(s1 + (cellV-v1)/(v2-v1)*(s2-s1))
		}
	}
	return 0
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }

func loadOptions(path string) Options {
	o := Options{Host: "192.168.0.119", Port: 65102, CanID: -1, CellCount: 20, ScanInterval: 60, MQTTPort: 1883}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Keine Optionen unter %s (%v), nutze Standardwerte", path, err)
		return o
	}
	if err := json.Unmarshal(data, &o); err != nil {
		log.Fatalf("Optionen ungültig: %v", err)
	}
	if o.CellCount < 1 {
		o.CellCount = 20
	}
	if o.ScanInterval < 5 {
		o.ScanInterval = 5
	}
	return o
}

// Publisher stellt die Werte in Home Assistant bereit.
type Publisher interface {
	Publish(v Values) error
	Name() string
}

func main() {
	optsPath := flag.String("options", "/data/options.json", "Pfad zur Optionsdatei")
	once := flag.Bool("once", false, "Einmal lesen, ausgeben und beenden (Test)")
	flag.Parse()
	log.SetFlags(log.LstdFlags)

	o := loadOptions(*optsPath)
	log.Printf("VESC Express %s:%d, CAN-ID %d (-1 = auto), %dS, Intervall %ds",
		o.Host, o.Port, o.CanID, o.CellCount, o.ScanInterval)

	read := func() (Values, error) {
		v, err := ReadVESC(o.Host, o.Port, o.CanID)
		if err != nil {
			return v, err
		}
		v.Voltage = round1(v.Voltage)
		v.TempESC = round1(v.TempESC)
		v.TempMotor = round1(v.TempMotor)
		v.SOC = calcSOC(v.Voltage, o.CellCount)
		return v, nil
	}

	if *once {
		v, err := read()
		if err != nil {
			log.Fatal(err)
		}
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
		return
	}

	pub, err := newPublisher(o)
	if err != nil {
		log.Fatalf("Kein Weg zu Home Assistant: %v", err)
	}
	log.Printf("Veröffentliche über %s", pub.Name())

	tick := time.NewTicker(time.Duration(o.ScanInterval) * time.Second)
	defer tick.Stop()
	for {
		v, err := read()
		if err != nil {
			log.Printf("VESC Fehler: %v", err)
		} else {
			log.Printf("%.1f V | SOC %.1f %% | ESC %.1f °C | Motor %.1f °C (CAN %d)",
				v.Voltage, v.SOC, v.TempESC, v.TempMotor, v.CanID)
			if err := pub.Publish(v); err != nil {
				log.Printf("Fehler beim Veröffentlichen: %v", err)
			}
		}
		<-tick.C
	}
}

// newPublisher: MQTT (eigene Optionen oder Supervisor-MQTT-Dienst),
// sonst Fallback auf die Home Assistant REST-API.
func newPublisher(o Options) (Publisher, error) {
	cfg, err := mqttConfig(o)
	if err == nil {
		return newMQTTPublisher(cfg)
	}
	log.Printf("Kein MQTT verfügbar (%v) – nutze Home Assistant REST-API", err)
	return newRESTPublisher()
}
