package video

type publicationEvent struct {
	ContentURI   string        `json:"contentUri"`
	Payload      *videoPayload `json:"payload"`
	LastModified string        `json:"lastModified"`
}

type identifier struct {
	Authority       string `json:"authority"`
	IdentifierValue string `json:"identifierValue"`
}

type brand struct {
	ID string `json:"id"`
}

type videoPayload struct {
	ID                    string       `json:"uuid,omitempty"`
	Title                 string       `json:"title,omitempty"`
	Standfirst            string       `json:"standfirst,omitempty"`
	Description           string       `json:"description,omitempty"`
	Byline                string       `json:"byline,omitempty"`
	Identifiers           []identifier `json:"identifiers,omitempty"`
	Brands                []brand      `json:"brands,omitempty"`
	FirstPublishedDate    string       `json:"firstPublishedDate,omitempty"`
	PublishedDate         string       `json:"publishedDate,omitempty"`
	MainImage             string       `json:"mainImage,omitempty"`
	StoryPackage          string       `json:"storyPackage,omitempty"`
	Transcript            string       `json:"transcript,omitempty"`
	Captions              []caption    `json:"captions,omitempty"`
	DataSources           []dataSource `json:"dataSource,omitempty"`
	CanBeDistributed      string       `json:"canBeDistributed,omitempty"`
	Type                  string       `json:"type,omitempty"`
	LastModified          string       `json:"lastModified,omitempty"`
	PublishReference      string       `json:"publishReference,omitempty"`
	CanBeSyndicated       string       `json:"canBeSyndicated,omitempty"`
	AccessLevel           string       `json:"accessLevel,omitempty"`
	WebURL                string       `json:"webUrl,omitempty"`
	CanonicalWebURL       string       `json:"canonicalWebUrl,omitempty"`
	PromotionalTitle      string       `json:"promotionalTitle,omitempty"`
	PromotionalStandfirst string       `json:"promotionalStandfirst,omitempty"`
}

type caption struct {
	Url       string `json:"url"`
	MediaType string `json:"mediaType"`
}

type dataSource struct {
	BinaryUrl   string   `json:"binaryUrl,omitempty"`
	PixelWidth  *float64 `json:"pixelWidth,omitempty"`
	PixelHeight *float64 `json:"pixelHeight,omitempty"`
	MediaType   string   `json:"mediaType,omitempty"`
	Duration    *float64 `json:"duration,omitempty"`
	VideoCodec  string   `json:"videoCodec,omitempty"`
	AudioCodec  string   `json:"audioCodec,omitempty"`
}
