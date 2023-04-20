package video

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	tid "github.com/Financial-Times/transactionid-utils-go"
)

type VideoMapperHandler struct {
	messageProducer    messageProducer
	messageTransformer messageTransformer
	log                *logger.UPPLogger
}

type messageProducer interface {
	SendMessage(kafka.FTMessage) error
}

type messageTransformer interface {
	TransformMsg(kafka.FTMessage) (kafka.FTMessage, string, error)
}

func NewRequestHandler(messageProducer messageProducer, messageTransformer messageTransformer, log *logger.UPPLogger) *VideoMapperHandler {
	return &VideoMapperHandler{
		messageProducer:    messageProducer,
		messageTransformer: messageTransformer,
		log:                log,
	}
}

func (v *VideoMapperHandler) OnMessage(m kafka.FTMessage) {
	transactionID := m.Headers["X-Request-Id"]
	if m.Headers["Origin-System-Id"] != systemOrigin {
		v.log.WithTransactionID(transactionID).
			WithField("Origin-System-Id", m.Headers["Origin-System-Id"]).
			Info("Ignoring message with different Origin-System-Id")
		return
	}
	contentType := m.Headers["Content-Type"]
	if strings.Contains(contentType, "application/json") {
		videoMsg, contentUUID, err := v.messageTransformer.TransformMsg(m)
		if err != nil {
			v.log.WithTransactionID(transactionID).
				WithError(err).
				Errorf("Error consuming message")
			return
		}
		err = v.messageProducer.SendMessage(videoMsg)
		if err != nil {
			v.log.WithTransactionID(transactionID).
				WithError(err).
				Error("Error sending transformed message to queue")
			return
		}
		v.log.WithTransactionID(transactionID).
			Infof("Mapped and sent for uuid: %v", contentUUID)
	} else {
		v.log.WithTransactionID(transactionID).
			Infof("Ignoring message with contentType %v", contentType)
		return
	}
}

func (v *VideoMapperHandler) MapRequest(w http.ResponseWriter, r *http.Request) {
	transactionID := tid.GetTransactionIDFromRequest(r)
	v.log.WithTransactionID(transactionID).Info("Received transformation request")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writerBadRequest(w, err, v.log)
	}

	m := createConsumerMessageFromRequest(transactionID, body, r)
	videoMsg, _, err := v.messageTransformer.TransformMsg(m)
	if err != nil {
		v.log.WithError(err).Error("Failed to transform message")
		writerBadRequest(w, err, v.log)
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write([]byte(videoMsg.Body))
	if err != nil {
		v.log.WithTransactionID(transactionID).
			WithError(err).
			Warn("Writing response error")
	}
}

func createConsumerMessageFromRequest(tid string, body []byte, r *http.Request) kafka.FTMessage {
	return kafka.FTMessage{
		Body: string(body),
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Request-Id":      tid,
			"Message-Timestamp": r.Header.Get("Message-Timestamp"),
		},
	}
}

func writerBadRequest(w http.ResponseWriter, err error, log *logger.UPPLogger) {
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		log.WithError(err).Warn("Couldn't write Bad Request response.")
	}
}
