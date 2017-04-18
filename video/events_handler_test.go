package video

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/upp-next-video-mapper/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type mockMessageProducer struct {
	message    string
	sendCalled bool
}

var mockMsgProducer mockMessageProducer
var eventsHandler VideoMapperHandler

func init() {
	InitLogs(os.Stdout, os.Stdout, os.Stderr)
}

func TestOnMessage_InvalidSystemId(t *testing.T) {
	m := consumer.Message{
		Headers: map[string]string{
			"Origin-System-Id": "jdhyhjsjsjsj",
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
	assert.Equal(t, videoOutput, mockMsgProducer.message)
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
	mockMsgProducer := &mockMessageProducer{}
	var msgProducer producer.MessageProducer = mockMsgProducer
	eventsHandler := &VideoMapperHandler{&msgProducer, VideoMapper{}}
	return eventsHandler, mockMsgProducer
}
