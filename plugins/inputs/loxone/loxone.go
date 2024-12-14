package loxone

import (
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
	Url  string       `toml:"url"`
	User string       `toml:"user"`
	Key  string       `toml:"key"`
	Item []LoxoneItem `toml:"item"`

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
	if loglevel == "DEBUG" {
		log.SetLevel(log.DebugLevel)
	} else if loglevel == "INFO" {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	var err error
	w.conn, err = loxone.New(
		loxone.WithHost(w.Url),
		loxone.WithUsernameAndPassword(w.User, w.Key),
		loxone.WithAutoReconnect(true),
		loxone.WithRegisterEvents(),
	)

	if err != nil {
		return err
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

			w.messages = append(w.messages, loxEvent)
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

		// Create a map for tags and include bucket in the destinationdb
		tags := map[string]string{
			"destinationdb": msg.mappedconfig.Bucket,
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
