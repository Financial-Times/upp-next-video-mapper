package video

import (
	"encoding/json"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/stretchr/testify/assert"
)

const (
	messageTimestamp = "2017-04-13T10:27:32.353Z"
	xRequestId       = "tid_123123"
)

var mapper = VideoMapper{log: logger.NewUPPLogger("video-mapper", "Debug")}

func TestTransformMsg_TidHeaderMissing(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.EqualError(t, err, "header X-Request-Id not found in kafka message headers. Skipping message", "Expected error when X-Request-Id is missing")
}

func TestTransformMsg_MessageTimestampHeaderMissing(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id": xRequestId,
		},
		Body: `{
			"id": "77fff607-bc22-450d-8c5d-e26fe1f0dc7c" 
		}`,
	}

	msg, _, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected when Message-Timestamp header is missing")
	assert.NotEmpty(t, msg.Body, "Message body should not be empty")
	assert.Contains(t, msg.Body, "\"lastModified\":", "LastModified field should be generated if header value is missing")
}

func TestTransformMsg_InvalidJson(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{{
					"lastModified": "2017-04-04T14:42:58.920Z",
					"publishReference": "tid_123123",
					"type": "video",
					"id": "bad50c54-76d9-30e9-8734-b999c708aa4c"}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.Error(t, err, "Expected error when invalid JSON for video content")
	assert.Contains(t, err.Error(), "Video JSON couldn't be unmarshalled. Skipping invalid JSON:", "Expected error message when invalid JSON for video content")
}

func TestTransformMsg_UuidMissing(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id": xRequestId,
		},
		Body: `{}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.Error(t, err, "Expected error when video UUID is missing")
	assert.Contains(t, err.Error(), "Could not extract UUID from video message. Skipping invalid JSON:", "Expected error when video UUID is missing")
}

func TestTransformMsg_UnpublishEvent(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{
					"deleted": true,
					"lastModified": "2017-04-04T14:42:58.920Z",
					"publishReference": "tid_123123",
					"type": "video",
					"uuid": "bad50c54-76d9-30e9-8734-b999c708aa4c"}`,
	}

	resultMsg, uuid, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected for unpublish event")
	assert.Equal(t, "bad50c54-76d9-30e9-8734-b999c708aa4c", uuid, "UUID not extracted correctly from unpublish event")
	assert.Equal(t, "{\"contentUri\":\"http://next-video-mapper.svc.ft.com/video/model/bad50c54-76d9-30e9-8734-b999c708aa4c\",\"payload\":{\"uuid\":\"bad50c54-76d9-30e9-8734-b999c708aa4c\",\"deleted\":true},\"lastModified\":\"2017-04-13T10:27:32.353Z\"}", resultMsg.Body)
}

func TestTransformMsg_Success(t *testing.T) {
	videoInput, err := readContent("video-input.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Input data for test cannot be loaded from external file")
	}
	videoOutput, err := readContent("video-output.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Output data for test cannot be loaded from external file")
	}

	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: videoInput,
	}

	resultMsg, _, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected for publish event")

	videoOutputStruct, resultMsgStruct, err := MapStringToPublicationEvent(videoOutput, resultMsg.Body)
	if assert.NoError(t, err, "Error mapping string") {
		assert.Equal(t, videoOutputStruct, resultMsgStruct)
	}
}

func TestTransformMsg_WithStoryPackage(t *testing.T) {
	var message = kafka.FTMessage{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{
					"id": "a40808ac-1417-4c48-9781-1dd2d8c8c6dc",
					"title": "ECB and Fed debates hit dollar and euro",
					"byline": "Filmed by Nicola Stansfield. Produced by Vanessa Kortekaas.",
					"description": "The FT's Katie Martin highlights the main stories in the markets.",
					"firstPublishedAt": "2017-04-06T09:58:35.440Z",
					"publishedAt": "2017-04-12T12:29:48.331Z",
					"related": [
						{
							"id": "494e4386-1a6e-11e7-a266-12672483791a"
						}
					]
				}`,
	}

	resultMsg, _, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected for unpublish event")
	assert.Contains(t, resultMsg.Body, "\"storyPackage\":\"a40808ac-1417-4c48-2945-63c109d95533\"")
}

func MapStringToPublicationEvent(videoOutput, retMsgBody string) (videoOutputStruct, resultMsgStruct *publicationEvent, err error) {
	videoOutputStruct = &publicationEvent{}
	resultMsgStruct = &publicationEvent{}
	if err := json.Unmarshal([]byte(videoOutput), videoOutputStruct); err != nil {
		return nil, nil, err
	}

	if err := json.Unmarshal([]byte(retMsgBody), resultMsgStruct); err != nil {
		return nil, nil, err
	}
	return videoOutputStruct, resultMsgStruct, nil
}
