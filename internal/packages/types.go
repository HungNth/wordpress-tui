package packages

type PackageRef struct {
	Type string `json:"type"` // "plugin" or "theme"
	Slug string `json:"slug"`
}

type Metadata struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Homepage       string `json:"homepage"`
	Author         string `json:"author"`
	AuthorHomepage string `json:"author_homepage"`
	Slug           string `json:"slug"`
	Type           string `json:"type"`
	LastUpdated    string `json:"last_updated"`
	Size           string `json:"size"`
	DownloadURL    string `json:"download_url"`
}
type Artifact struct {
	Ref         PackageRef
	Version     string
	Path        string
	Size        int64
	SHA256      string
	IsCached    bool
	IsStale     bool
	StaleReason string
}
