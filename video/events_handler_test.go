package video

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v3"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type mockMessageProducer struct {
	message    string
	sendCalled bool
}

func TestNewVideoMapperHandler(t *testing.T) {
	log := logger.NewUPPLogger("next-video-mapper", "Debug")
	handler := NewRequestHandler(&mockMessageProducer{
		message:    "Test",
		sendCalled: false,
	}, VideoMapper{log: log}, log)
	assert.NotNil(t, handler.messageProducer, "Message producer should be set")
	assert.NotNil(t, handler.messageTransformer, "Message transformer should be set")
}

func TestMapHandler_InvalidBody(t *testing.T) {
	req, err := http.NewRequest("POST", "/map", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()

	requestHandler, _ := createRequestHandler()

	r := mux.NewRouter()
	r.HandleFunc("/map", requestHandler.MapRequest).Methods("POST")
	r.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code, "Unexpected status code")
}

func TestOnMessage_InvalidSystemId(t *testing.T) {
	m := kafka.FTMessage{
		Headers: map[string]string{
			"Origin-System-Id": "hasfsafaf",
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createRequestHandler()
	eventsHandler.OnMessage(m)
	assert.Equal(t, false, mockMsgProducer.sendCalled, "Producer message not expected to be generated when invalid Origin Id")
}

func TestOnMessage_SkipAudioContent(t *testing.T) {
	m := kafka.FTMessage{
		Headers: map[string]string{
			"Origin-System-Id": "http://cmdb.ft.com/systems/next-video-editor",
			"Content-Type":     "application/vnd.ft-upp-audio+json",
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createRequestHandler()
	eventsHandler.OnMessage(m)
	assert.Equal(t, false, mockMsgProducer.sendCalled, "Producer message not expected to be generated when invalid Origin Id")
}

func TestOnMessage_MappingError(t *testing.T) {
	m := kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Origin-System-Id":  systemOrigin,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{}`,
	}

	eventsHandler, mockMsgProducer := createRequestHandler()
	eventsHandler.OnMessage(m)

	assert.Equal(t, false, mockMsgProducer.sendCalled, "Error expected when mapping fails")
}

func TestOnMessage_Success(t *testing.T) {
	videoInput, err := readContent("video-input.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Input data for test cannot be loaded from external file")
	}

	m := kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Origin-System-Id":  systemOrigin,
			"Message-Timestamp": messageTimestamp,
			"Content-Type":      "application/json",
		},
		Body: videoInput,
	}

	eventsHandler, mockMsgProducer := createRequestHandler()
	eventsHandler.OnMessage(m)
	assert.Exactly(t, true, mockMsgProducer.sendCalled, "Mapped video content should be produced")

	videoOutput, err := readContent("video-output.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Output data for test cannot be loaded from external file")
	}

	videoOutputStruct, resultMsgStruct, err := MapStringToPublicationEvent(videoOutput, mockMsgProducer.message)
	if assert.NoError(t, err, "Error mapping string") {
		assert.Equal(t, videoOutputStruct, resultMsgStruct)
	}
}

func (mock *mockMessageProducer) SendMessage(message kafka.FTMessage) error {
	mock.message = message.Body
	mock.sendCalled = true
	return nil
}

func (mock *mockMessageProducer) ConnectivityCheck() (string, error) {
	// do nothing
	return "", nil
}

func createRequestHandler() (*VideoMapperHandler, *mockMessageProducer) {
	mockMsgProducer := mockMessageProducer{}
	msgProducer := &mockMsgProducer
	log := logger.NewUPPLogger("video-mapper", "Debug")

	return NewRequestHandler(msgProducer, VideoMapper{log: log}, log), &mockMsgProducer
}

func readContent(fileName string) (string, error) {
	data, err := os.ReadFile("test-resources/" + fileName)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
