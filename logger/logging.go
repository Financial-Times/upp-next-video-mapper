package logger

import "github.com/Sirupsen/logrus"

type AppLogger struct {
	Log         *logrus.Logger
	ServiceName string
}

func NewAppLogger(serviceName string) *AppLogger {
	logrus.SetLevel(logrus.InfoLevel)
	log := logrus.New()
	log.Formatter = new(logrus.JSONFormatter)

	return &AppLogger{Log: log, ServiceName: serviceName}
}

func (appLogger *AppLogger) ServiceStartedEvent(port int) {
	event := make(map[string]interface{})
	event["service_name"] = appLogger.ServiceName
	event["event"] = "service_started"
	appLogger.Log.WithFields(event).Infof("Starting to listen on port [%d]", port)
}

func (appLogger *AppLogger) QueueConsumerStarted(queueTopic string) {
	event := make(map[string]interface{})
	event["service_name"] = appLogger.ServiceName
	event["event"] = "consume_queue"
	appLogger.Log.WithFields(event).Infof("Starting queue consumer: %v", queueTopic)
}

func (appLogger *AppLogger) InfoMessageEvent(message string, transactionID string, contentUUID string) {
	event := make(map[string]interface{})
	event["service_name"] = appLogger.ServiceName
	event["event"] = "mapping"
	event["transaction_id"] = transactionID
	if contentUUID != "" {
		event["uuid"] = contentUUID
	}

	appLogger.Log.WithFields(event).Info(message)
}

func (appLogger *AppLogger) WarnMessageEvent(message string, transactionID string, contentUUID string, err error) {
	event := make(map[string]interface{})
	event["event"] = "error"
	event["service_name"] = appLogger.ServiceName

	if err != nil {
		event["error"] = err
	}
	if transactionID != "" {
		event["transaction_id"] = transactionID
	}
	if contentUUID != "" {
		event["uuid"] = contentUUID
	}

	appLogger.Log.WithFields(event).Warn(message)
}

func (appLogger *AppLogger) ErrorMessageEvent(message string, transactionID string, contentUUID string, err error) {
	event := make(map[string]interface{})
	event["event"] = "error"
	event["service_name"] = appLogger.ServiceName
	event["error"] = err

	if transactionID != "" {
		event["transaction_id"] = transactionID
	}
	if contentUUID != "" {
		event["uuid"] = contentUUID
	}

	appLogger.Log.WithFields(event).Error(message)
}
