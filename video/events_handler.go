package video

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/service-status-go/httphandlers"
	tid "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"
)

const videoSystemOrigin = "http://cmdb.ft.com/systems/next-video-editor"

type VideoMapperHandler struct {
	messageProducer producer.MessageProducer
	videoMapper     VideoMapper
}

func NewVideoMapperHandler(producerConfig producer.MessageProducerConfig) VideoMapperHandler {
	videoMapper := VideoMapper{}
	messageProducer := producer.NewMessageProducer(producerConfig)

	return VideoMapperHandler{messageProducer, videoMapper}
}

func (v *VideoMapperHandler) Listen(hc *Healthcheck, port int) {
	r := mux.NewRouter()
	r.HandleFunc("/map", v.MapHandler).Methods("POST")
	r.HandleFunc("/__health", fthealth.Handler(hc.Healthcheck()))
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler)
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler)

	http.Handle("/", r)
	Logger.ServiceStartedEvent(port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		Logger.ErrorMessageEvent("Couldn't set up HTTP listener", "", "", err)
	}
}

func (v *VideoMapperHandler) OnMessage(m consumer.Message) {
	transactionID := m.Headers["X-Request-Id"]
	if m.Headers["Origin-System-Id"] != videoSystemOrigin {
		Logger.InfoMessageEvent(fmt.Sprintf("Ignoring message with different Origin-System-Id %v", m.Headers["Origin-System-Id"]), transactionID, "")
		return
	}

	videoMsg, contentUUID, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		Logger.ErrorMessageEvent("Error consuming message: ", transactionID, contentUUID, err)
		return
	}
	err = (v.messageProducer).SendMessage("", videoMsg)
	if err != nil {
		Logger.ErrorMessageEvent("Error sending transformed message to queue", transactionID, contentUUID, err)
		return
	}
	Logger.InfoMessageEvent(fmt.Sprintf("Mapped and sent for uuid: %v", contentUUID), transactionID, contentUUID)
}

func (v *VideoMapperHandler) MapHandler(w http.ResponseWriter, r *http.Request) {
	transactionID := tid.GetTransactionIDFromRequest(r)
	Logger.InfoMessageEvent("Received transformation request", transactionID, "")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writerBadRequest(w, transactionID, err)
	}

	m := createConsumerMessageFromRequest(transactionID, body, r)
	videoMsg, contentUUID, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		Logger.ErrorMessageEvent("Error consuming message: ", transactionID, contentUUID, err)
		writerBadRequest(w, transactionID, err)
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write([]byte(videoMsg.Body))
	if err != nil {
		Logger.WarnMessageEvent("Error writing response", transactionID, contentUUID, err)
	}
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

func writerBadRequest(w http.ResponseWriter, transactionID string, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		Logger.WarnMessageEvent("Couldn't write Bad Request response.", transactionID, "", err)
	}
	return
}
