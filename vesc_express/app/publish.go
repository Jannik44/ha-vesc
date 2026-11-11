package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type sensorDef struct {
	Key, Name, Unit, DeviceClass, Icon string
}

var sensors = []sensorDef{
	{"voltage", "Akkuspannung", "V", "voltage", ""},
	{"soc", "Ladestand", "%", "battery", ""},
	{"temp_esc", "ESC Temperatur", "°C", "temperature", "mdi:chip"},
	{"temp_motor", "Motor Temperatur", "°C", "temperature", "mdi:engine"},
}

const (
	nodeID        = "vesc_express"
	stateTopic    = nodeID + "/state"
	availTopic    = nodeID + "/availability"
	haStatusTopic = "homeassistant/status"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func supervisorGet(path string, out any) error {
	token := os.Getenv("SUPERVISOR_TOKEN")
	if token == "" {
		return errors.New("SUPERVISOR_TOKEN nicht gesetzt")
	}
	req, _ := http.NewRequest("GET", "http://supervisor"+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supervisor %s: HTTP %d %s", path, resp.StatusCode, bytes.TrimSpace(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ---------------------------------------------------------------- MQTT

type mqttCfg struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

func mqttConfig(o Options) (mqttCfg, error) {
	if o.MQTTHost != "" {
		return mqttCfg{Host: o.MQTTHost, Port: o.MQTTPort, Username: o.MQTTUser, Password: o.MQTTPassword}, nil
	}
	var resp struct {
		Data mqttCfg `json:"data"`
	}
	if err := supervisorGet("/services/mqtt", &resp); err != nil {
		return mqttCfg{}, err
	}
	if resp.Data.Host == "" {
		return mqttCfg{}, errors.New("MQTT-Dienst liefert keinen Host")
	}
	return resp.Data, nil
}

type mqttPublisher struct {
	client mqtt.Client
	broker string
}

func newMQTTPublisher(cfg mqttCfg) (*mqttPublisher, error) {
	scheme := "tcp"
	if cfg.SSL {
		scheme = "ssl"
	}
	broker := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port)
	p := &mqttPublisher{broker: broker}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("vesc-express-addon").
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(10*time.Second).
		SetWill(availTopic, "offline", 1, true)
	opts.OnConnect = func(c mqtt.Client) {
		log.Printf("MQTT verbunden (%s)", broker)
		p.announce()
		// Discovery erneut senden, wenn Home Assistant neu startet
		c.Subscribe(haStatusTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
			if string(m.Payload()) == "online" {
				p.announce()
			}
		})
	}
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		log.Printf("MQTT Verbindung verloren: %v", err)
	}

	p.client = mqtt.NewClient(opts)
	t := p.client.Connect()
	if !t.WaitTimeout(15 * time.Second) {
		log.Printf("MQTT noch nicht verbunden, versuche es im Hintergrund weiter")
	} else if t.Error() != nil {
		return nil, t.Error()
	}
	return p, nil
}

func (p *mqttPublisher) Name() string { return "MQTT " + p.broker }

func (p *mqttPublisher) announce() {
	device := map[string]any{
		"identifiers":  []string{nodeID},
		"name":         "VESC",
		"manufacturer": "VESC",
		"model":        "VESC Express",
	}
	for _, s := range sensors {
		cfg := map[string]any{
			"name":                        s.Name,
			"unique_id":                   nodeID + "_" + s.Key,
			"object_id":                   "vesc_" + s.Key,
			"state_topic":                 stateTopic,
			"value_template":              "{{ value_json." + s.Key + " }}",
			"unit_of_measurement":         s.Unit,
			"device_class":                s.DeviceClass,
			"state_class":                 "measurement",
			"suggested_display_precision": 1,
			"availability_topic":          availTopic,
			"device":                      device,
		}
		if s.Icon != "" {
			cfg["icon"] = s.Icon
		}
		b, _ := json.Marshal(cfg)
		p.client.Publish(fmt.Sprintf("homeassistant/sensor/%s/%s/config", nodeID, s.Key), 1, true, b)
	}
	p.client.Publish(availTopic, 1, true, "online")
}

func (p *mqttPublisher) Publish(v Values) error {
	b, _ := json.Marshal(v)
	t := p.client.Publish(stateTopic, 1, true, b)
	if !t.WaitTimeout(10 * time.Second) {
		return errors.New("MQTT Publish Timeout")
	}
	return t.Error()
}

// ---------------------------------------------------------------- REST

type restPublisher struct{ token string }

func newRESTPublisher() (*restPublisher, error) {
	token := os.Getenv("SUPERVISOR_TOKEN")
	if token == "" {
		return nil, errors.New("SUPERVISOR_TOKEN nicht gesetzt")
	}
	return &restPublisher{token: token}, nil
}

func (p *restPublisher) Name() string { return "Home Assistant REST-API" }

func (p *restPublisher) Publish(v Values) error {
	vals := map[string]float64{
		"voltage": v.Voltage, "soc": v.SOC, "temp_esc": v.TempESC, "temp_motor": v.TempMotor,
	}
	for _, s := range sensors {
		attrs := map[string]any{
			"unit_of_measurement": s.Unit,
			"device_class":        s.DeviceClass,
			"state_class":         "measurement",
			"friendly_name":       "VESC " + s.Name,
		}
		if s.Icon != "" {
			attrs["icon"] = s.Icon
		}
		body, _ := json.Marshal(map[string]any{"state": vals[s.Key], "attributes": attrs})
		req, _ := http.NewRequest("POST", "http://supervisor/core/api/states/sensor.vesc_"+s.Key, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+p.token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			return fmt.Errorf("sensor.vesc_%s: HTTP %d", s.Key, resp.StatusCode)
		}
	}
	return nil
}
