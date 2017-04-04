package video

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/next-video-mapper/logger"
	"io"
	"regexp"
	"strings"
)

const videoContentURIBase = "http://next-video-mapper.svc.ft.com/video/model/"
const videoAuthority = "http://api.ft.com/system/NEXT-VIDEO-EDITOR"
const ftBrandID = "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"

const publishedDate = "updatedAt"
const canBeDistributedYes = "yes"
const videoType = "MediaResource"

var (
	uuidExtractRegex = regexp.MustCompile(".*/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$")
)

type VideoMapper struct {
}

func (v VideoMapper) TransformMsg(m consumer.Message) (marshalledEvent []byte, uuid string, err error) {
	tid := m.Headers["X-Request-Id"]
	if tid == "" {
		return nil, "", errors.New("X-Request-Id not found in kafka message headers. Skipping message")
	}

	lastModified := m.Headers["Message-Timestamp"]
	if lastModified == "" {
		return nil, "", errors.New("Message-Timestamp not found in kafka message headers. Skipping message")
	}

	marshalledEvent, uuid, err = v.mapVideoContent(m, tid, lastModified)
	if err != nil {
		WarnLogger.Printf("%v - Mapping error: [%v]", tid, err.Error())
		return nil, "", err
	}
	return marshalledEvent, uuid, nil
}

func (v VideoMapper) mapVideoContent(m consumer.Message, tid string, lastModified string) ([]byte, string, error) {
	var videoContent map[string]interface{}

	if err := json.Unmarshal([]byte(m.Body), &videoContent); err != nil {
		return nil, "", fmt.Errorf("Video JSON couldn't be unmarshalled. Skipping invalid JSON: %v", m.Body)
	}

	uuid, err := get("id", videoContent)
	if err != nil {
		return nil, "", err
	}

	isPublishEvent, err := isPublishEvent(videoContent)
	if err != nil {
		return nil, uuid, err
	}

	//it's an unpublish event
	if !isPublishEvent {
		return mapUnpublishEvent(uuid)
	}

	contentURI := videoContentURIBase + uuid
	videoModel, err := getVideoModel(videoContent, uuid, tid)
	if err != nil {
		return nil, uuid, err
	}

	marshalledPubEvent, err := buildAndMarshalPublicationEvent(videoModel, contentURI, publishedDate, tid)
	return marshalledPubEvent, uuid, err
}

func mapUnpublishEvent(uuid string) ([]byte, string, error) {
	return nil, uuid, nil
}

func getVideoModel(videoContent map[string]interface{}, uuid string, tid string) (*videoPayload, error) {
	title, _ := get("title", videoContent)
	standfirst, _ := get("standfirst", videoContent)
	description, _ := get("description", videoContent)
	byline, _ := get("byline", videoContent)
	firstPublishDate, _ := get("createdAt", videoContent)

	publishedDate, err := getPublishedDate(videoContent)
	if err != nil {
		WarnLogger.Println(err)
	}

	mainImage, err := getMainImage(videoContent, tid, uuid)
	if err != nil {
		WarnLogger.Println(err)
	}

	transcriptionMap, transcript, err := getTranscript(videoContent, tid, uuid)
	if err != nil {
		WarnLogger.Println(err)
	}

	captionsList := getCaptions(transcriptionMap)
	dataSources := getDataSources(videoContent["encoding"], tid)

	i := identifier{
		Authority:       videoAuthority,
		IdentifierValue: uuid,
	}

	b := brand{
		ID: ftBrandID,
	}

	videoModel := &videoPayload{
		Id:                 uuid,
		Title:              title,
		Standfirst:         standfirst,
		Description:        description,
		Byline:             byline,
		Identifiers:        []identifier{i},
		Brands:             []brand{b},
		FirstPublishedDate: firstPublishDate,
		PublishedDate:      publishedDate,
		MainImage:          mainImage,
		Transcript:         transcript,
		Captions:           captionsList,
		DataSources:        dataSources,
		CanBeDistributed:   canBeDistributedYes,
		Type:               videoType,
	}

	return videoModel, nil
}

func getPublishedDate(video map[string]interface{}) (string, error) {
	updatedAt, err1 := get("updatedAt", video)
	if err1 == nil {
		return updatedAt, nil
	}
	createdAt, err2 := get("createdAt", video)
	if err2 == nil {
		return createdAt, nil
	}

	return "", fmt.Errorf("No valid value could be found for publishedDate: [%v] [%v]", err1, err2)
}

func getMainImage(videoContent map[string]interface{}, tid string, uuid string) (string, error) {
	imageURI, err := get("image", videoContent)
	if err != nil {
		return "", err
	}

	imageUuidString, err := getUUIDFromURI(imageURI)
	WarnLogger.Printf("Image uuid: %v", imageUuidString)
	if err != nil {
		return "", err
	}

	imageUuid, err := NewUUIDFromString(imageUuidString)
	if err != nil {
		return "", err
	}

	mainImageSetUuid, err := GenerateImageSetUUID(*imageUuid)
	if err != nil {
		return "", err
	}

	return mainImageSetUuid.String(), nil
}

func getTranscript(videoContent map[string]interface{}, tid string, uuid string) (map[string]interface{}, string, error) {
	transcription, ok := videoContent["transcription"]
	if !ok {
		return nil, "", fmt.Errorf("%v - Transcription is null and will be skipped for uuid: %v", tid, uuid)
	}

	transcriptionMap, ok := transcription.(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("%v - Transcription is null and will be skipped for uuid: %v", tid, uuid)
	}

	transcript, err := get("transcript", transcriptionMap)
	if err != nil {
		return transcriptionMap, "", err
	}

	valid := isValidXHTML(transcript)
	if !valid {
		return transcriptionMap, "", fmt.Errorf("%v - Transcription has invalid HTML body and will be skipped for uuid: %v", tid, uuid)
	}

	return transcriptionMap, transcript, nil
}

func getCaptions(transcriptionMap map[string]interface{}) []caption {
	cList := []caption{}
	if transcriptionMap == nil {
		return cList
	}

	captions, _ := transcriptionMap["captions"].([]interface{})
	for _, elem := range captions {
		captionMap := elem.(map[string]interface{})
		mediaType, _ := get("mediaType", captionMap)
		url, _ := get("url", captionMap)

		c := caption{
			Url:       url,
			MediaType: mediaType,
		}
		cList = append(cList, c)
	}

	return cList
}

func getDataSources(encoding interface{}, tid string) []dataSource {
	encodingMap, ok := encoding.(map[string]interface{})
	if !ok {
		InfoLogger.Printf("%v - encoding field of video JSON is null, dataSource will be empty.", tid)
	}

	outputs, ok := encodingMap["outputs"]
	if !ok {
		InfoLogger.Printf("%v - outputs field of video JSON is null, dataSource will be empty.", tid)
		return nil
	}

	outputsArray := outputs.([]interface{})
	dataSourcesList := []dataSource{}
	for _, elem := range outputsArray {
		elemMap, okMap := elem.(map[string]interface{})
		if !okMap {
			InfoLogger.Printf("%v - cannot extract output field of video JSON, dataSource will be empty.", tid)
		}
		binaryUrl, _ := get("url", elemMap)
		pWidth, _ := getNumber("width", elemMap)
		pHeight, _ := getNumber("height", elemMap)
		mediaType, _ := get("mediaType", elemMap)
		videoCodec, _ := get("videoCodec", elemMap)
		audioCodec, _ := get("audioCodec", elemMap)
		duration, _ := getNumber("duration", elemMap)

		d := dataSource{
			BinaryUrl:   binaryUrl,
			PixelWidth:  pWidth,
			PixelHeight: pHeight,
			MediaType:   mediaType,
			Duration:    duration,
			VideoCodec:  videoCodec,
			AudioCondec: audioCodec,
		}

		dataSourcesList = append(dataSourcesList, d)
	}

	return dataSourcesList
}

func buildAndMarshalPublicationEvent(p *videoPayload, contentURI, lastModified, pubRef string) (_ []byte, err error) {
	e := publicationEvent{
		ContentURI:   contentURI,
		Payload:      p,
		LastModified: lastModified,
	}

	marshalledEvent, err := unsafeJSONMarshal(e)
	if err != nil {
		WarnLogger.Printf("%v - Couldn't marshall event %v, skipping message.", pubRef, e)
		return nil, err
	}

	return marshalledEvent, nil
}

func isPublishEvent(video map[string]interface{}) (bool, error) {
	_, err := getPublishedDate(video)
	if err == nil {
		return true, nil
	}
	if _, present := video["deleted"]; present {
		//it's an unpublish event
		return false, nil
	}
	return false, fmt.Errorf("Could not detect event type for video: [%#v]", video)
}

func getUUIDFromURI(uri string) (string, error) {
	result := uuidExtractRegex.FindStringSubmatch(uri)
	if len(result) == 2 {
		return result[1], nil
	}
	return "", fmt.Errorf("Couldn't extract uuid from uri %s", uri)
}

func getPrefixedUrl(prefix string, uuid string) string {
	return prefix + uuid
}

func get(key string, videoContent map[string]interface{}) (val string, _ error) {
	valueI, ok := videoContent[key]
	if !ok {
		return "", fmt.Errorf("[%s] field of native video JSON is null", key)
	}

	val, ok = valueI.(string)
	if !ok {
		return "", fmt.Errorf("[%s] field of native video JSON is not a string", key)
	}
	return val, nil
}

func getNumber(key string, inputMap map[string]interface{}) (*float64, error) {
	valueI, ok := inputMap[key]
	if !ok {
		return nil, fmt.Errorf("[%s] field of native video JSON is null", key)
	}

	val, isOk := valueI.(float64)
	if !isOk {
		return nil, fmt.Errorf("[%s] field of native video JSON is not a number", key)
	}

	return &val, nil
}

func isValidXHTML(data string) bool {
	d := xml.NewDecoder(strings.NewReader(data))

	valid := true
	for {
		_, err := d.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			valid = false
			break
		}
	}

	return valid
}

func unsafeJSONMarshal(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)

	return b, nil
}
