package video

import (
	"encoding/json"
	"errors"
	"fmt"
	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/upp-next-video-mapper/logger"
	"io/ioutil"
	"net/http"
)

type Healthcheck struct {
	Client       http.Client
	ConsumerConf consumer.QueueConfig
}

func (h *Healthcheck) Healthcheck() *fthealth.HealthCheck {
	checks := []fthealth.Check{h.messageQueueProxyReachable()}
	return &fthealth.HealthCheck{SystemCode: "upp-next-video-mapper", Name: "Next Video Mapper", Description: "Checks if all the dependent services are reachable and healthy.", Checks: checks}
}

func (h *Healthcheck) messageQueueProxyReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Publishing or updating videos will not be possible, clients will not see the new content.",
		Name:             "MessageQueueProxyReachable",
		PanicGuide:       "https://dewey.ft.com/up-vm.html",
		Severity:         1,
		TechnicalSummary: "Message queue proxy is not reachable/healthy",
		Checker:          h.checkAggregateMessageQueueProxiesReachable,
	}

}

func (h *Healthcheck) checkAggregateMessageQueueProxiesReachable() (string, error) {
	errMsg := ""
	for i := 0; i < len(h.ConsumerConf.Addrs); i++ {
		err := h.checkMessageQueueProxyReachable(h.ConsumerConf.Addrs[i], h.ConsumerConf.Topic, h.ConsumerConf.AuthorizationKey, h.ConsumerConf.Queue)
		if err == nil {
			return "", nil
		}
		errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", h.ConsumerConf.Addrs[i], err.Error())
	}
	return errMsg, errors.New(errMsg)
}

func (h *Healthcheck) checkMessageQueueProxyReachable(address string, topic string, authKey string, queue string) error {
	req, err := http.NewRequest("GET", address+"/topics", nil)
	if err != nil {
		WarnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}
	if len(authKey) > 0 {
		req.Header.Add("Authorization", authKey)
	}
	if len(queue) > 0 {
		req.Host = queue
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		WarnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Proxy returned status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}
	body, err := ioutil.ReadAll(resp.Body)
	return checkIfTopicIsPresent(body, topic)
}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string
	err := json.Unmarshal(body, &topics)
	if err != nil {
		return fmt.Errorf("Error occured and topic could not be found. %v", err.Error())
	}
	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}
	return errors.New("Topic was not found")
}
