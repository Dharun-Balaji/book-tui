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
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	CoverURL      string   `json:"coverURL"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Tags          []string `json:"tags"`
	TotalChapters int      `json:"totalChapters"`
}

type Chapter struct {
	URL    string  `json:"url"`
	Title  string  `json:"title"`
	Number float64 `json:"number"`
}

type SearchResult struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	CoverURL string `json:"coverURL"`
	Status   string `json:"status"`
}

type SelectorMatch struct {
	Text  string            `json:"text"`
	HTML  string            `json:"html"`
	Attrs map[string]string `json:"attrs"`
}
