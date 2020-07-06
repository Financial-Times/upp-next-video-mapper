package video

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockMessageProducer struct {
	message    string
	sendCalled bool
}

func TestNewVideoMapperHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	cfg := producer.MessageProducerConfig{
		Addr:          "localhost:3000",
		Topic:         "writeTopic",
		Queue:         "writeQueue",
		Authorization: "authorization",
	}

	handler := NewVideoMapperHandler(cfg, s.Client())
	assert.NotNil(t, handler.messageProducer, "Message producer should be set")
	assert.NotNil(t, handler.videoMapper, "Video mapper should be set")
}

func TestMapHandler_InvalidBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/map", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()

	eventsHandler, _ := createEventsHandler()

	r := mux.NewRouter()
	r.HandleFunc("/map", eventsHandler.MapHandler).Methods("POST")
	r.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code, "Unexpected status code")
}

func TestOnMessage_InvalidSystemId(t *testing.T) {
	m := consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "hasfsafaf",
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createEventsHandler()
	eventsHandler.OnMessage(m)
	assert.Equal(t, false, mockMsgProducer.sendCalled, "Producer message not expected to be generated when invalid Origin Id")
}

func TestOnMessage_SkipAudioContent(t *testing.T) {
	m := consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
			"Content-Type":     "application/vnd.ft-upp-audio+json",
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createEventsHandler()
	eventsHandler.OnMessage(m)
	assert.Equal(t, false, mockMsgProducer.sendCalled, "Producer message not expected to be generated when invalid Origin Id")
}

func TestOnMessage_MappingError(t *testing.T) {
	m := consumer.Message{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Origin-System-Id":  videoSystemOrigin,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createEventsHandler()
	eventsHandler.OnMessage(m)

	assert.Equal(t, false, mockMsgProducer.sendCalled, "Error expected when mapping fails")
}

func TestOnMessage_Success(t *testing.T) {
	videoInput, err := readContent("video-input.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Input data for test cannot be loaded from external file")
	}

	m := consumer.Message{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Origin-System-Id":  videoSystemOrigin,
			"Message-Timestamp": messageTimestamp,
			"Content-Type":      "application/json",
		},
		Body: videoInput,
	}

	eventsHandler, mockMsgProducer := createEventsHandler()
	eventsHandler.OnMessage(m)
	assert.Exactly(t, true, mockMsgProducer.sendCalled, "Mapped video content should be produced")

	videoOutput, err := readContent("video-output.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Output data for test cannot be loaded from external file")
	}

	videoOutputStruct, resultMsgStruct, err := mapStringToPublicationEvent(videoOutput, mockMsgProducer.message)
	if assert.NoError(t, err, "Error mapping string") {
		assert.Equal(t, videoOutputStruct, resultMsgStruct)
	}
}

func (mock *mockMessageProducer) SendMessage(uuid string, message producer.Message) error {
	mock.message = message.Body
	mock.sendCalled = true
	return nil
}

func (mock *mockMessageProducer) ConnectivityCheck() (string, error) {
	// do nothing
	return "", nil
}

func createEventsHandler() (*VideoMapperHandler, *mockMessageProducer) {
	var msgProducer producer.MessageProducer
	var mockMsgProducer mockMessageProducer

	mockMsgProducer = mockMessageProducer{}
	msgProducer = &mockMsgProducer

	return &VideoMapperHandler{msgProducer, VideoMapper{}}, &mockMsgProducer
}
