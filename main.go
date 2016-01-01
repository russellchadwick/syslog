package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/russellchadwick/event"
	"github.com/russellchadwick/messagebroker"
	"gopkg.in/mcuadros/go-syslog.v2"
)

var (
	syslogAddress = flag.String("syslog-address", "0.0.0.0:514", "address to listen for remote syslog")
	dbHost        = flag.String("db-host", "localhost", "host for postgres database")
	dbDatabase    = flag.String("db-database", "messagebroker", "database name for postgres database")
	dbUser        = flag.String("db-user", "messagebroker", "user for postgres database")
	dbPassword    = flag.String("db-password", "messagebroker", "password for postgres database")
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
		log.Fatalln("error listening on syslog address: ", err)
	}
	err = server.Boot()
	if err != nil {
		log.Fatalln("error starting syslog server: ", err)
	}
	defer server.Kill()

	meter := metrics.NewMeter()
	err = metrics.Register("incoming_message_meter", meter)
	if err != nil {
		log.Fatalln("error creating metric: ", err)
	}

	config := messagebroker.PostgresqlConnectionConfig{
		Host:     *dbHost,
		Database: *dbDatabase,
		User:     *dbUser,
		Password: *dbPassword,
	}
	messageBroker, err := messagebroker.NewPostgresqlMessageBroker(&config)
	if err != nil {
		log.Fatalln("error creating message broker: ", err)
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {

			syslogEvent := convertLogPartsToSyslogEvent(logParts)

			syslogEventBytes, err := json.Marshal(syslogEvent)
			if err != nil {
				log.Println("error marshaling event: ", err)
			}

			err = messageBroker.Publish("Syslog.1", syslogEventBytes)
			if err != nil {
				log.Fatalln("error sending event: ", err)
			}

			fmt.Println(logParts)

			meter.Mark(1)
		}
	}(channel)

	go metrics.Log(metrics.DefaultRegistry, time.Duration(1)*time.Minute, log.New(os.Stdout, "metrics: ", log.Lmicroseconds))

	fmt.Println("Waiting")
	server.Wait()

}

func convertLogPartsToSyslogEvent(logParts map[string]interface{}) *event.SyslogV1 {

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
