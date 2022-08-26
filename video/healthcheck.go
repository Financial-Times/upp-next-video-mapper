package video

import (
	"fmt"
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/service-status-go/gtg"
)

const (
	ResponseOK = "OK"
)

type HealthCheck struct {
	consumer      messageConsumerHealthcheck
	producer      messageProducerHealthcheck
	appName       string
	appSystemCode string
}

type messageProducerHealthcheck interface {
	ConnectivityCheck() error
}

type messageConsumerHealthcheck interface {
	ConnectivityCheck() error
	MonitorCheck() error
}

func NewHealthCheck(p messageProducerHealthcheck, c messageConsumerHealthcheck, appName string, appSystemCode string) *HealthCheck {
	return &HealthCheck{
		consumer:      c,
		producer:      p,
		appName:       appName,
		appSystemCode: appSystemCode,
	}
}

func (h *HealthCheck) Health() func(w http.ResponseWriter, r *http.Request) {
	checks := []fthealth.Check{h.readQueueCheck(), h.readQueueLagCheck(), h.writeQueueCheck()}
	hc := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  h.appSystemCode,
			Name:        h.appName,
			Description: "Checks if all the dependent services are reachable and healthy.",
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}
	return fthealth.Handler(hc)
}

func (h *HealthCheck) readQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "read-message-queue-reachable",
		Name:             "Read Message Queue Reachable",
		Severity:         2,
		BusinessImpact:   "Publishing or updating videos will not be possible, clients will not see the new content.",
		TechnicalSummary: "Read message queue is not reachable/healthy",
		PanicGuide:       fmt.Sprintf("https://runbooks.ftops.tech/%s", h.appSystemCode),
		Checker:          h.checkIfKafkaIsReachableFromConsumer,
	}
}

func (h *HealthCheck) writeQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "write-message-queue-reachable",
		Name:             "Write Message Queue Reachable",
		Severity:         2,
		BusinessImpact:   "Publishing or updating videos will not be possible, clients will not see the new content.",
		TechnicalSummary: "Write message queue is not reachable/healthy",
		PanicGuide:       fmt.Sprintf("https://runbooks.ftops.tech/%s", h.appSystemCode),
		Checker:          h.checkIfKafkaIsReachableFromProducer,
	}
}

func (h *HealthCheck) readQueueLagCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "read-message-queue-lagging",
		Name:             "Read Message Queue Is Not Lagging",
		Severity:         3,
		BusinessImpact:   "Publishing or updating videos will be possible with latency.",
		TechnicalSummary: "Messages awaiting handling exceed the configured lag tolerance. Check if Kafka consumer is stuck.",
		PanicGuide:       fmt.Sprintf("https://runbooks.ftops.tech/%s", h.appSystemCode),
		Checker:          h.CheckIfConsumerIsLagging,
	}
}

func (h *HealthCheck) GTG() gtg.Status {
	consumerCheck := func() gtg.Status {
		return gtgCheck(h.checkIfKafkaIsReachableFromConsumer)
	}
	producerCheck := func() gtg.Status {
		return gtgCheck(h.checkIfKafkaIsReachableFromProducer)
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{
		consumerCheck,
		producerCheck,
	})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func (h *HealthCheck) checkIfKafkaIsReachableFromConsumer() (string, error) {
	err := h.consumer.ConnectivityCheck()
	if err != nil {
		return "", err
	}
	return ResponseOK, nil
}

func (h *HealthCheck) CheckIfConsumerIsLagging() (string, error) {
	err := h.consumer.MonitorCheck()
	if err != nil {
		return "", err
	}
	return ResponseOK, nil
}

func (h *HealthCheck) checkIfKafkaIsReachableFromProducer() (string, error) {
	err := h.producer.ConnectivityCheck()
	if err != nil {
		return "", err
	}
	return ResponseOK, nil
}
