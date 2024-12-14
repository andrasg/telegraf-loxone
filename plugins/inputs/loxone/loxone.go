package loxone

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/XciD/loxone-ws"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	log "github.com/sirupsen/logrus"
)

// LoxoneInput is the main struct for the plugin
type LoxoneInput struct {
	Url        string       `toml:"url"`
	User       string       `toml:"user"`
	Key        string       `toml:"key"`
	Item       []LoxoneItem `toml:"item"`
	JsonConfig string       `toml:"jsonconfig"`

	conn     loxone.Loxone
	messages []LoxoneEvent
	mu       sync.Mutex
}

// LoxoneEvent represents a single event from Loxone
type LoxoneEvent struct {
	uuid         string
	value        float64
	time         time.Time
	mappedconfig LoxoneItem
}

// LoxoneItem represents a single item to be collected
type LoxoneItem struct {
	Bucket      string            `toml:"bucket"`
	Field       string            `toml:"field"`
	UUID        string            `toml:"uuid"`
	Tags        map[string]string `toml:"tags"`
	Measurement string            `toml:"measurement"`
}

type PointMapping struct {
	Bucket       string        `json:"bucket"`
	Measurements []Measurement `json:"measurements"`
}

type Measurement struct {
	Name       string      `json:"name"`
	Datapoints []Datapoint `json:"datapoints"`
}

type Datapoint struct {
	Fields     map[string]string `json:"fields"`
	Tags       map[string]string `json:"tags"`
	IsCritical bool              `json:"isCritical,omitempty"`
}

func ReadPointMappings(filename string) ([]PointMapping, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var data struct {
		PointMappings []PointMapping `json:"pointMappings"`
	}
	if err := json.Unmarshal(byteValue, &data); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return data.PointMappings, nil
}

func (w *LoxoneInput) Description() string {
	return "Reads metrics from Loxone"
}

func (w *LoxoneInput) SampleConfig() string {
	return `
  ## WebSocket server URL
  url = "192.168.1.253"

  ## Username for authentication
  user = "user"

  ## password for authentication
  key = "pass"

  ## definition of items to be collected
  [[inputs.loxone.item]]
    bucket = "loxone"
    measurement = "measurement"
    uuid = "122c6fd0-0056-abde-ffff796b564594c0"

    ## optional tags
    [inputs.loxone.item.tags]
      room = "roomname"
`
}

func (w *LoxoneInput) findItemByUUID(uuid string) (LoxoneItem, bool) {
	for _, configitem := range w.Item {
		if configitem.UUID == uuid {
			// set default field to "value" if not set
			if configitem.Field == "" {
				configitem.Field = "value"
			}

			return configitem, true
		}
	}
	return LoxoneItem{}, false
}

func (w *LoxoneInput) Start(acc telegraf.Accumulator) error {
	loglevel := os.Getenv("LOGLEVEL")

	switch loglevel {
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}

	var err error
	w.conn, err = loxone.New(
		loxone.WithHost(w.Url),
		loxone.WithUsernameAndPassword(w.User, w.Key),
		loxone.WithAutoReconnect(true),
		loxone.WithRegisterEvents(),
		loxone.WithKeepAliveInterval(30*time.Second),
	)

	if err != nil {
		return err
	}

	if w.JsonConfig != "" {
		mappings, _ := ReadPointMappings(w.JsonConfig)
		log.Debugf("Read json mappings: %v", len(mappings))

		items := []LoxoneItem{}

		for _, mapping := range mappings {

			log.Debugf("Mapping: %v", mapping.Bucket)
			log.Debugf("  Number of measurements: %v", len(mapping.Measurements))

			for _, measurement := range mapping.Measurements {
				log.Debugf("  Measurement: %v", measurement.Name)
				log.Debugf("    Number of datapoints: %v", len(measurement.Datapoints))

				for i, datapoint := range measurement.Datapoints {
					log.Debugf("    Datapoint: %v", i)
					log.Debugf("      Number of tags: %v", len(datapoint.Tags))

					tags := map[string]string{}

					for tag, tagvalue := range datapoint.Tags {
						log.Debugf("        Tag: %v=%v", tag, tagvalue)
						tags[tag] = datapoint.Tags[tag]
					}

					log.Debugf("      Number of fields: %v", len(datapoint.Fields))
					for field, uuid := range datapoint.Fields {
						log.Debugf("        Field: %v", field)
						log.Debugf("        UUID: %v", uuid)

						item := LoxoneItem{
							Bucket:      mapping.Bucket,
							Measurement: measurement.Name,
							Field:       field,
							UUID:        uuid,
							Tags:        tags,
						}

						items = append(items, item)
					}
				}
			}
		}

		// append config
		w.Item = append(w.Item, items...)
	}

	go func() {
		for event := range w.conn.GetEvents() {

			// check if we need to handle the event
			item, found := w.findItemByUUID(event.UUID)
			if !found {
				continue
			}

			loxEvent := LoxoneEvent{
				uuid:         event.UUID,
				value:        event.Value,
				time:         time.Now(),
				mappedconfig: item,
			}

			log.Infof("Received event: %v, %v, %v=%v",
				loxEvent.mappedconfig.Bucket,
				loxEvent.mappedconfig.Measurement,
				loxEvent.mappedconfig.Field,
				loxEvent.value)

			w.mu.Lock()
			w.messages = append(w.messages, loxEvent)
			w.mu.Unlock()
		}
	}()
	return nil
}

func (w *LoxoneInput) Gather(acc telegraf.Accumulator) error {
	w.mu.Lock()
	messages := w.messages
	w.messages = nil
	w.mu.Unlock()

	for _, msg := range messages {
		fields := make(map[string]interface{})
		fields[msg.mappedconfig.Field] = msg.value

		// Create a map for tags
		tags := map[string]string{}

		// include the bucket as tag if set
		if msg.mappedconfig.Bucket != "" {
			tags["bucket"] = msg.mappedconfig.Bucket
		}

		// Add tags from the config
		for tagname := range msg.mappedconfig.Tags {
			tags[tagname] = msg.mappedconfig.Tags[tagname]
		}

		acc.AddFields(msg.mappedconfig.Measurement, fields, tags, msg.time)
	}
	return nil
}

func (w *LoxoneInput) Stop() {
	if w.conn != nil && w.conn.IsConnected() {
		log.Println("Closing Loxone connection")
		w.conn.Close()
	}
}

func init() {
	inputs.Add("loxone", func() telegraf.Input {
		return &LoxoneInput{}
	})
}
