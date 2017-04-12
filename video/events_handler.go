package video

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/next-video-mapper/logger"
	tid "github.com/Financial-Times/transactionid-utils-go"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"strconv"
)

const videoSystemOrigin = "http://cmdb.ft.com/systems/next-video-editor"

type VideoMapperHandler struct {
	messageProducer *producer.MessageProducer
	videoMapper     VideoMapper
}

func NewVideoMapperHandler(producerConfig producer.MessageProducerConfig) VideoMapperHandler {
	videoMapper := VideoMapper{}
	messageProducer := producer.NewMessageProducer(producerConfig)

	return VideoMapperHandler{&messageProducer, videoMapper}
}

func (v VideoMapperHandler) Listen(hc *Healthcheck, port int) {
	r := mux.NewRouter()
	r.HandleFunc("/map", v.MapHandler).Methods("POST")
	r.HandleFunc("/__health", hc.Healthcheck()).Methods("GET")
	r.HandleFunc("/__gtg", hc.gtg).Methods("GET")

	http.Handle("/", r)
	InfoLogger.Printf("Starting to listen on port [%d]", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		ErrorLogger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func (v VideoMapperHandler) OnMessage(m consumer.Message) {
	transactionID := m.Headers["X-Request-Id"]
	if m.Headers["Origin-System-Id"] != videoSystemOrigin {
		InfoLogger.Printf("%v - Ignoring message with different Origin-System-Id %v", transactionID, m.Headers["Origin-System-Id"])
		return
	}

	videoMsg, contentUUID, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		WarnLogger.Printf("%v - Error consuming message: %v", transactionID, err)
		return
	}
	err = (*v.messageProducer).SendMessage("", videoMsg)
	if err != nil {
		WarnLogger.Printf("%v - Error sending transformed message to queue: %v", transactionID, err)
	}
	InfoLogger.Printf("%v - Mapped and sent for uuid: %v", transactionID, contentUUID)
}

func (v VideoMapperHandler) MapHandler(w http.ResponseWriter, r *http.Request) {
	transactionID := tid.GetTransactionIDFromRequest(r)
	InfoLogger.Printf("%v - Received transformation request", transactionID)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writerBadRequest(w, err)
	}

	m := createMessageFromRequest(transactionID, body, r)
	videoMsg, _, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		writerBadRequest(w, err)
	}

	w.Header().Add("Content-Type", "application/json")
	_, err = w.Write([]byte(videoMsg.Body))

	if err != nil {
		WarnLogger.Printf("%v - Writing response error: [%v]", transactionID, err)
	}
}

func createMessageFromRequest(tid string, body []byte, r *http.Request) consumer.Message {
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
	_, err2 := w.Write([]byte(err.Error()))
	if err2 != nil {
		WarnLogger.Printf("Couldn't write Bad Request response. %v", err2)
	}
	return
}
