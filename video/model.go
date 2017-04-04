package video

type publicationEvent struct {
	ContentURI   string        `json:"contentUri"`
	Payload      *videoPayload `json:"payload,omitempty"`
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
	Id                 string       `json:"uuid"`
	Title              string       `json:"title,omitempty"`
	Standfirst         string       `json:"standfirst,omitempty"`
	Description        string       `json:"description,omitempty"`
	Byline             string       `json:"byline,omitempty"`
	Identifiers        []identifier `json:"identifiers"`
	Brands             []brand      `json:"brands"`
	FirstPublishedDate string       `json:"firstPublishedDate"`
	PublishedDate      string       `json:"publishedDate"`
	MainImage          string       `json:"image,omitempty"`
	Transcript         string       `json:"transcript,omitempty"`
	Captions           []caption    `json:"captions,omitempty"`
	DataSources        []dataSource `json:"dataSource,omitempty"`
	CanBeDistributed   string       `json:"canBeDistributed"`
	Type               string       `json:"type"`
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
	AudioCondec string   `json:"audioCodec,omitempty"`
}
