package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Financial-Times/kafka-client-go/v4"
	cli "github.com/jawher/mow.cli"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/upp-next-video-mapper/video"
	"github.com/gorilla/mux"
)

const (
	serviceName        = "next-video-mapper"
	serviceDescription = "Catch native video content transform into Content and send back to queue."
)

func main() {
	app := cli.App(serviceName, serviceDescription)

	appName := app.String(cli.StringOpt{
		Name:   "app-name",
		Value:  "Next Video Mapper",
		Desc:   "The name of the application",
		EnvVar: "APP_NAME",
	})

	appSystemCode := app.String(cli.StringOpt{
		Name:   "app-system-code",
		Value:  "next-video-mapper",
		Desc:   "App system code",
		EnvVar: "APP_SYSTEM_CODE",
	})

	kafkaAddress := app.String(cli.StringOpt{
		Name:   "queue-kafkaAddress",
		Value:  "",
		Desc:   "Address to connect to kafka.",
		EnvVar: "KAFKA_ADDRESS",
	})

	group := app.String(cli.StringOpt{
		Name:   "group",
		Desc:   "Group used to read the messages from the queue.",
		EnvVar: "Q_GROUP",
	})

	readTopic := app.String(cli.StringOpt{
		Name:   "read-topic",
		Desc:   "The topic to read the messages from.",
		EnvVar: "Q_READ_TOPIC",
	})

	writeTopic := app.String(cli.StringOpt{
		Name:   "write-topic",
		Desc:   "The topic to write the messages to.",
		EnvVar: "Q_WRITE_TOPIC",
	})

	appPort := app.Int(cli.IntOpt{
		Name:   "port",
		Value:  8080,
		Desc:   "Application port to listen on",
		EnvVar: "APP_PORT",
	})

	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Logging level (DEBUG, INFO, WARN, ERROR)",
		EnvVar: "LOG_LEVEL",
	})

	consumerLagTolerance := app.Int(cli.IntOpt{
		Name:   "consumerLagTolerance",
		Value:  120,
		Desc:   "Kafka lag tolerance",
		EnvVar: "KAFKA_LAG_TOLERANCE",
	})

	clusterArn := app.String(cli.StringOpt{
		Name:   "kafka-cluster-arn",
		Desc:   "Amazon Resource Name for the kafka cluster",
		EnvVar: "KAFKA_CLUSTER_ARN",
	})

	log := logger.NewUPPLogger(serviceName, *logLevel)

	log.Infof("[Startup] %s is starting", serviceName)

	app.Action = func() {

		if *kafkaAddress == "" {
			log.Fatal("No queue kafkaAddress provided. Quitting...")
		}

		producerConfig := kafka.ProducerConfig{
			ClusterArn:              clusterArn,
			BrokersConnectionString: *kafkaAddress,
			Topic:                   *writeTopic,
		}
		producer, err := kafka.NewProducer(producerConfig)
		defer func(producer *kafka.Producer) {
			err := producer.Close()
			if err != nil {
				log.WithError(err).Error("Producer could not stop")
			}
		}(producer)

		if err != nil {
			log.WithError(err).Fatal("Failed to create Kafka producer")
		}

		consumerConfig := kafka.ConsumerConfig{
			ClusterArn:              clusterArn,
			BrokersConnectionString: *kafkaAddress,
			ConsumerGroup:           *group,
		}

		topics := []*kafka.Topic{
			kafka.NewTopic(*readTopic, kafka.WithLagTolerance(int64(*consumerLagTolerance))),
		}
		videoMapper := video.NewVideoMapper(log)
		handler := video.NewRequestHandler(producer, videoMapper, log)
		log.Info(prettyPrintConfig(consumerConfig, producerConfig, *readTopic))

		consumer, err := kafka.NewConsumer(consumerConfig, topics, log)

		if err != nil {
			log.WithError(err).Fatal("Failed to create Kafka consumer")
		}

		go consumer.Start(handler.OnMessage)
		defer func(consumer *kafka.Consumer) {
			err = consumer.Close()
			if err != nil {
				log.WithError(err).Error("Consumer could not stop")
			}
		}(consumer)

		hc := video.NewHealthCheck(producer, consumer, *appName, *appSystemCode)
		go serveEndpoints(handler, hc, *appPort, log)
		waitForSignal()
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func serveEndpoints(serviceHandler *video.VideoMapperHandler, hc *video.HealthCheck, port int, log *logger.UPPLogger) {
	r := mux.NewRouter()
	r.HandleFunc("/map", serviceHandler.MapRequest).Methods("POST")
	r.HandleFunc("/__health", hc.Health())
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler)
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler)
	r.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG))

	http.Handle("/", r)
	log.Infof("Starting to listen on port [%d]", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		log.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}

func prettyPrintConfig(c kafka.ConsumerConfig, p kafka.ProducerConfig, readTopic string) string {
	return fmt.Sprintf("Config: [\n\t%s\n\t%s\n]", prettyPrintConsumerConfig(c, readTopic), prettyPrintProducerConfig(p))
}

func prettyPrintConsumerConfig(c kafka.ConsumerConfig, readTopic string) string {
	return fmt.Sprintf("consumerConfig: [\n\t\taddr: [%v]\n\t\tgroup: [%v]\n\t\ttopic: [%v]\n\t\t]", c.BrokersConnectionString, c.ConsumerGroup, readTopic)
}

func prettyPrintProducerConfig(p kafka.ProducerConfig) string {
	return fmt.Sprintf("producerConfig: [\n\t\taddr: [%v]\n\t\ttopic: [%v]\n\t\t]", p.BrokersConnectionString, p.Topic)
}
