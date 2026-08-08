package source

type Metadata struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	BaseURL   string `json:"baseURL"`
	Language  string `json:"language"`
	NeedsJS   bool   `json:"needsJS"`
	RateLimit int    `json:"rateLimit"`
}

type Novel struct {
	URL, Title, Author, CoverURL, Description, Status string
	Tags                                              []string
	TotalChapters                                     int
}

type Chapter struct {
	URL, Title string
	Number     float64
}

type SearchResult struct {
	URL, Title, Author, CoverURL, Status string
}

type SelectorMatch struct {
	Text  string            `json:"text"`
	HTML  string            `json:"html"`
	Attrs map[string]string `json:"attrs"`
}
