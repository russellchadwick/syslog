package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jeromer/syslogparser"
	"github.com/rcrowley/go-metrics"
	"github.com/russellchadwick/event"
	"github.com/russellchadwick/eventsource"
	"gopkg.in/mcuadros/go-syslog.v2"
)

var (
	syslogAddress    = flag.String("syslog-address", "0.0.0.0:514", "address to listen for remote syslog")
	connectionString = flag.String("connection-string", "dbname=event user=event password=event", "connecting string for database to store events")
)

func main() {

	flag.Parse()

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC3164)
	server.SetHandler(handler)
	err := server.ListenUDP(*syslogAddress)
	if err != nil {
		log.Fatalf("error listening on syslog address: %s \n", err)
	}
	err = server.Boot()
	if err != nil {
		log.Fatalf("error starting syslog server: %s \n", err)
	}
	defer server.Kill()

	meter := metrics.NewMeter()
	err = metrics.Register("incoming_message_meter", meter)
	if err != nil {
		log.Fatalf("error creating metric: %s \n", err)
	}

	eventStore, err := eventsource.NewPostgresqlEventStore(*connectionString)
	if err != nil {
		log.Fatalf("error creating event source: %s \n", err)
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {

			syslogEvent := convertLogPartsToSyslogEvent(logParts)

			id, err := eventStore.Send("Syslog.1", syslogEvent)
			if err != nil {
				log.Fatalf("error sending event: %s \n", err)
			}

			fmt.Println(id)
			fmt.Println(logParts)

			meter.Mark(1)
		}
	}(channel)

	go metrics.Log(metrics.DefaultRegistry, time.Duration(1)*time.Minute, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))

	fmt.Println("Waiting")
	server.Wait()

}

func convertLogPartsToSyslogEvent(logParts syslogparser.LogParts) *event.SyslogV1 {

	client, ok := logParts["client"].(string)
	if !ok {
		log.Fatalf("error reading client as a string")
	}
	content, ok := logParts["content"].(string)
	if !ok {
		log.Fatalf("error reading content as a string")
	}
	facility, ok := logParts["facility"].(int)
	if !ok {
		log.Fatalf("error reading facility as a number")
	}
	hostname, ok := logParts["hostname"].(string)
	if !ok {
		log.Fatalf("error reading hostname as a string")
	}
	priority, ok := logParts["priority"].(int)
	if !ok {
		log.Fatalf("error reading priority as a number")
	}
	severity, ok := logParts["severity"].(int)
	if !ok {
		log.Fatalf("error reading severity as a number")
	}
	tag, ok := logParts["tag"].(string)
	if !ok {
		log.Fatalf("error reading tag as a string")
	}
	timestamp, ok := logParts["timestamp"].(time.Time)
	if !ok {
		log.Fatalf("error reading timstamp as a Time.time")
	}

	syslogEvent := &event.SyslogV1{
		Client:    client,
		Content:   content,
		Facility:  facility,
		Hostname:  hostname,
		Priority:  priority,
		Severity:  severity,
		Tag:       tag,
		Timestamp: timestamp,
	}

	return syslogEvent
}
