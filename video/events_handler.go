package video

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	consumer "github.com/Financial-Times/message-queue-gonsumer"
	"github.com/Financial-Times/service-status-go/httphandlers"
	tid "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
)

const videoSystemOrigin = "http://cmdb.ft.com/systems/next-video-editor"

type VideoMapperHandler struct {
	messageProducer producer.MessageProducer
	videoMapper     VideoMapper
}

func NewVideoMapperHandler(producerConfig producer.MessageProducerConfig, client *http.Client) VideoMapperHandler {
	videoMapper := VideoMapper{}
	messageProducer := producer.NewMessageProducerWithHTTPClient(producerConfig, client)

	return VideoMapperHandler{messageProducer, videoMapper}
}

func (v *VideoMapperHandler) Listen(hc *HealthCheck, port int) {
	r := mux.NewRouter()
	r.HandleFunc("/map", v.MapHandler).Methods("POST")
	r.HandleFunc("/__health", hc.Health())
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler)
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler)
	r.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG))

	http.Handle("/", r)
	logger.Infof("Starting to listen on port [%d]", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		logger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func (v *VideoMapperHandler) OnMessage(m consumer.Message) {
	transactionID := m.Headers["X-Request-Id"]
	if m.Headers["Origin-System-Id"] != videoSystemOrigin {
		logger.Infof("%v - Ignoring message with different Origin-System-Id %v", transactionID, m.Headers["Origin-System-Id"])
		return
	}
	contentType := m.Headers["Content-Type"]
	if strings.Contains(contentType, "application/json") {
		videoMsg, contentUUID, err := v.videoMapper.TransformMsg(m)
		if err != nil {
			logger.Errorf("%v - Error consuming message: %v", transactionID, err)
			return
		}
		err = (v.messageProducer).SendMessage("", videoMsg)
		if err != nil {
			logger.Errorf("%v - Error sending transformed message to queue: %v", transactionID, err)
			return
		}
		logger.Infof("%v - Mapped and sent for uuid: %v", transactionID, contentUUID)
	} else {
		logger.Infof("%v - Ignoring message with contentType %v", transactionID, contentType)
		return
	}
}

func (v *VideoMapperHandler) MapHandler(w http.ResponseWriter, r *http.Request) {
	transactionID := tid.GetTransactionIDFromRequest(r)
	logger.Infof("%v - Received transformation request", transactionID)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writerBadRequest(w, err)
	}

	m := createConsumerMessageFromRequest(transactionID, body, r)
	videoMsg, _, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		logger.Errorln(err)
		writerBadRequest(w, err)
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write([]byte(videoMsg.Body))
	if err != nil {
		logger.Warnf("%v - Writing response error: [%v]", transactionID, err)
	}
}

func (v *VideoMapperHandler) GetProducer() producer.MessageProducer {
	return v.messageProducer
}

func createConsumerMessageFromRequest(tid string, body []byte, r *http.Request) consumer.Message {
	return consumer.Message{
		Body: string(body),
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Request-Id":      tid,
			"Message-Timestamp": r.Header.Get("Message-Timestamp"),
		},
	}
}

func writerBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		logger.Warnf("Couldn't write Bad Request response. %v", err)
	}
	return
}
