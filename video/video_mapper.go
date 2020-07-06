package video

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	uuidUtils "github.com/Financial-Times/uuid-utils-go"
	"github.com/satori/go.uuid"

	"github.com/Financial-Times/go-logger"
)

const (
	canBeDistributedYes     = "yes"
	videoType               = "Video"
	videoContentURIBase     = "http://next-video-mapper.svc.ft.com/video/model/"
	videoAuthority          = "http://api.ft.com/system/NEXT-VIDEO-EDITOR"
	ftBrandID               = "http://api.ft.com/things/dbb0bdae-1f0c-11e4-b0cb-b2227cce2b54"
	dateFormat              = "2006-01-02T15:04:05.000Z0700"
	defaultAccessLevel      = "free"
	uuidGenerationSalt      = "storypackage"
	webUrlTemplate          = "https://www.ft.com/content/%s"
	canonicalWebUrlTemplate = "https://www.ft.com/content/%s"
)

var uuidExtractRegex = regexp.MustCompile(".*/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$")

type VideoMapper struct {
}

func (v VideoMapper) TransformMsg(m consumer.Message) (msg producer.Message, uuid string, err error) {
	tid := m.Headers["X-Request-Id"]
	if tid == "" {
		return producer.Message{}, "", fmt.Errorf("header X-Request-Id not found in kafka message headers. Skipping message")
	}

	lastModified := m.Headers["Message-Timestamp"]
	if lastModified == "" {
		lastModified = time.Now().Format(dateFormat)
	}

	var videoContent map[string]interface{}
	if err := json.Unmarshal([]byte(m.Body), &videoContent); err != nil {
		return producer.Message{}, "", fmt.Errorf("Error: %v - Video JSON couldn't be unmarshalled. Skipping invalid JSON: %v", err.Error(), m.Body)
	}

	uuid, err = get("id", videoContent)
	if err != nil {
		return producer.Message{}, "", fmt.Errorf("Error: %v - Could not extract UUID from video message. Skipping invalid JSON: %v", err.Error(), m.Body)
	}

	contentURI := getPrefixedUrl(videoContentURIBase, uuid)
	isPublishEvent := isPublishEvent(videoContent)

	//it's an unpublish event
	if !isPublishEvent {
		videoModel := &videoPayload{}
		deleteVideoMsg, err := buildAndMarshalPublicationEvent(videoModel, contentURI, lastModified, tid)
		return deleteVideoMsg, uuid, err
	}

	videoModel := getVideoModel(videoContent, uuid, tid, lastModified)
	videoMsg, err := buildAndMarshalPublicationEvent(videoModel, contentURI, lastModified, tid)
	return videoMsg, uuid, err
}

func getVideoModel(videoContent map[string]interface{}, uuid string, tid string, lastModified string) *videoPayload {
	title, _ := get("title", videoContent)
	standfirst, _ := get("standfirst", videoContent)
	description, _ := get("description", videoContent)
	byline, _ := get("byline", videoContent)
	firstPublishDate, _ := get("firstPublishedAt", videoContent)
	publishedDate, _ := get("publishedAt", videoContent)
	promotionalTitle, _ := get("promotionalTitle", videoContent)
	promotionalStandfirst, _ := get("promotionalStandfirst", videoContent)

	mainImage, err := getMainImage(videoContent)
	if err != nil {
		logger.Warnf("%v - Extract main image: %v", tid, err)
	}

	storyPackageUuid, err := getStoryPackageUUID(videoContent, uuid)
	if err != nil {
		logger.Warnf("%v - Extract story package: %v", tid, err)
	}

	transcriptionMap, transcript, err := getTranscript(videoContent, uuid)
	if err != nil {
		logger.Warnf("%v - %v", tid, err)
	}

	captionsList := getCaptions(transcriptionMap)
	dataSources, err := getDataSources(videoContent["encoding"])
	if err != nil {
		logger.Warnf("%v - %v", tid, err)
	}

	canBeSyndicated := getCanBeSyndicated(videoContent, tid)

	i := identifier{
		Authority:       videoAuthority,
		IdentifierValue: uuid,
	}

	b := brand{
		ID: ftBrandID,
	}

	accessLevel := getAccessLevel()

	webURL := fmt.Sprintf(webUrlTemplate, uuid)
	canonicalWebURL := fmt.Sprintf(canonicalWebUrlTemplate, uuid)

	return &videoPayload{
		ID:                    uuid,
		Title:                 title,
		Standfirst:            standfirst,
		Description:           description,
		Byline:                byline,
		Identifiers:           []identifier{i},
		Brands:                []brand{b},
		FirstPublishedDate:    firstPublishDate,
		PublishedDate:         publishedDate,
		MainImage:             mainImage,
		StoryPackage:          storyPackageUuid,
		Transcript:            transcript,
		Captions:              captionsList,
		DataSources:           dataSources,
		CanBeDistributed:      canBeDistributedYes,
		Type:                  videoType,
		LastModified:          lastModified,
		PublishReference:      tid,
		CanBeSyndicated:       canBeSyndicated,
		AccessLevel:           accessLevel,
		WebURL:                webURL,
		CanonicalWebURL:       canonicalWebURL,
		PromotionalTitle:      promotionalTitle,
		PromotionalStandfirst: promotionalStandfirst,
	}
}
func getCanBeSyndicated(videoContent map[string]interface{}, tid string) string {
	canBeSyndicated, err := getBool("canBeSyndicated", videoContent)
	if err != nil {
		logger.Warnf("%v - %v. Defaulting value to true", tid, err)
		canBeSyndicated = true
	}
	switch canBeSyndicated {
	case false:
		return "no"
	default:
		return "yes"
	}
}

func getMainImage(videoContent map[string]interface{}) (string, error) {
	imageURI, err := get("image", videoContent)
	if err != nil {
		return "", err
	}

	imageUUIDString, err := getUUIDFromURI(imageURI)
	if err != nil {
		return "", err
	}

	imageUUID, _ := uuidUtils.NewUUIDFromString(imageUUIDString)
	uuidDeriver := uuidUtils.NewUUIDDeriverWith(uuidUtils.IMAGE_SET)
	mainImageSetUUID, err := uuidDeriver.From(imageUUID)
	if err != nil {
		return "", err
	}

	return mainImageSetUUID.String(), nil
}

func getStoryPackageUUID(videoContent map[string]interface{}, videoUUID string) (string, error) {
	_, ok := videoContent["related"]
	if !ok {
		return "", fmt.Errorf("Related content is null and will be skipped for uuid: %v", videoUUID)
	}

	vUUID, err := uuidUtils.NewUUIDFromString(videoUUID)
	if err != nil {
		return "", err
	}

	uuidDeriver := uuidUtils.NewUUIDDeriverWith(uuidGenerationSalt)
	storyPackageUUID, err := uuidDeriver.From(vUUID)
	if err != nil {
		return "", err
	}

	return storyPackageUUID.String(), nil
}

func getTranscript(videoContent map[string]interface{}, uuid string) (map[string]interface{}, string, error) {
	transcription, ok := videoContent["transcription"]
	if !ok {
		return nil, "", fmt.Errorf("Transcription is null and will be skipped for uuid: %v", uuid)
	}

	transcriptionMap, ok := transcription.(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("Transcription is null and will be skipped for uuid: %v", uuid)
	}

	transcript, err := get("transcript", transcriptionMap)
	if err != nil {
		return transcriptionMap, "", err
	}

	valid := isValidXHTML(transcript)
	if !valid {
		return transcriptionMap, "", fmt.Errorf("Transcription has invalid HTML body and will be skipped for uuid: %v", uuid)
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

func getDataSources(encoding interface{}) ([]dataSource, error) {
	encodingMap, ok := encoding.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Encodings field of video JSON is null, dataSource will be empty.")
	}

	outputs, ok := encodingMap["outputs"]
	if !ok {
		return nil, fmt.Errorf("Outputs field of video JSON is null, dataSource will be empty.")
	}

	outputsArray := outputs.([]interface{})
	dataSourcesList := []dataSource{}
	for _, elem := range outputsArray {
		elemMap, okMap := elem.(map[string]interface{})
		if !okMap {
			return nil, fmt.Errorf("Cannot extract output field of video JSON, dataSource will be empty.")
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
			AudioCodec:  audioCodec,
		}

		dataSourcesList = append(dataSourcesList, d)
	}

	return dataSourcesList, nil
}

func getAccessLevel() string {
	return defaultAccessLevel
}

func buildAndMarshalPublicationEvent(p *videoPayload, contentURI, lastModified, pubRef string) (producer.Message, error) {
	e := publicationEvent{
		ContentURI:   contentURI,
		Payload:      p,
		LastModified: lastModified,
	}

	marshalledEvent, err := unsafeJSONMarshal(e)
	if err != nil {
		logger.Warnf("%v - Couldn't marshall event %v, skipping message.", pubRef, e)
		return producer.Message{}, err
	}

	headers := map[string]string{
		"X-Request-Id":      pubRef,
		"Message-Timestamp": lastModified,
		"Message-ID":        uuid.NewV4().String(),
		"Message-Type":      "cms-content-published",
		"Content-Type":      "application/json",
		"Origin-System-ID":  videoSystemOrigin,
	}
	return producer.Message{Headers: headers, Body: string(marshalledEvent)}, nil
}

func isPublishEvent(video map[string]interface{}) bool {
	if isDeleted, present := video["deleted"]; present {
		dFlag, ok := isDeleted.(bool)
		if ok && dFlag {
			return false
		}
	}
	return true
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

func getBool(key string, inputMap map[string]interface{}) (bool, error) {
	valueI, ok := inputMap[key]
	if !ok {
		return false, fmt.Errorf("[%s] field of native video JSON is null", key)
	}

	val, isOk := valueI.(bool)
	if !isOk {
		return false, fmt.Errorf("[%s] field of native video JSON is not a bool", key)
	}

	return val, nil
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
