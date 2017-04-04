package video

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/next-video-mapper/logger"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const videoSystemOrigin = "http://cmdb.ft.com/systems/next-video-editor"
const dateFormat = "2006-01-02T03:04:05.000Z0700"

type VideoMapperHandler struct {
	messageProducer *producer.MessageProducer
	videoMapper     VideoMapper
}

func NewVideoMapperHandler(producerConfig producer.MessageProducerConfig) VideoMapperHandler {
	videoMapper := VideoMapper{}
	messageProducer := producer.NewMessageProducer(producerConfig)

	return VideoMapperHandler{&messageProducer, videoMapper}
}

func (v VideoMapperHandler) Listen(hc *Healthcheck) {
	r := mux.NewRouter()
	r.HandleFunc("/map", v.MapHandler).Methods("POST")
	r.HandleFunc("/__health", hc.Healthcheck()).Methods("GET")
	r.HandleFunc("/__gtg", hc.gtg).Methods("GET")

	http.Handle("/", r)
	port := 8081 //hardcoded for now
	InfoLogger.Printf("Starting to listen on port [%d]", port)
	err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
	if err != nil {
		ErrorLogger.Panicf("Couldn't set up HTTP listener: %+v\n", err)
	}
}

func (v VideoMapperHandler) OnMessage(m consumer.Message) {
	tid := m.Headers["X-Request-Id"]
	if m.Headers["Origin-System-Id"] != videoSystemOrigin {
		InfoLogger.Printf("%v - Ignoring message with different Origin-System-Id %v", tid, m.Headers["Origin-System-Id"])
		return
	}

	marshalledEvent, contentUUID, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		WarnLogger.Printf("%v - Error error consuming message: %v", tid, err)
		return
	}
	headers := createHeader(tid)
	err = (*v.messageProducer).SendMessage("", producer.Message{Headers: headers, Body: string(marshalledEvent)})
	if err != nil {
		WarnLogger.Printf("%v - Error sending transformed message to queue: %v", tid, err)
	}
	InfoLogger.Printf("%v - Mapped and sent for uuid: %v", tid, contentUUID)
}

func (v VideoMapperHandler) MapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writerBadRequest(w, err)
	}
	tid := r.Header.Get("X-Request-Id")
	m := consumer.Message{
		Body:    string(body),
		Headers: createHeader(tid),
	}
	
	mappedVideoBytes, _, err := v.videoMapper.TransformMsg(m)
	if err != nil {
		writerBadRequest(w, err)
	}
	_, err = w.Write(mappedVideoBytes)
	if err != nil {
		WarnLogger.Printf("%v - Writing response error: [%v]", tid, err)
	}
}

func createHeader(tid string) map[string]string {
	return map[string]string{
		"X-Request-Id":      tid,
		"Message-Timestamp": time.Now().Format(dateFormat),
		"Message-Id":        uuid.NewV4().String(),
		"Message-Type":      "cms-content-published",
		"Content-Type":      "application/json",
		"Origin-System-Id":  "http://cmdb.ft.com/systems/brightcove",
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
