package packages

type PackageType string

const (
	PackageTypePlugin PackageType = "plugin"
	PackageTypeTheme  PackageType = "theme"
)

type PackageRef struct {
	Type PackageType `json:"type"`
	Slug string      `json:"slug"`
}

type Metadata struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	Homepage       string      `json:"homepage"`
	Author         string      `json:"author"`
	AuthorHomepage string      `json:"author_homepage"`
	Slug           string      `json:"slug"`
	Type           PackageType `json:"type"`
	LastUpdated    string      `json:"last_updated"`
	Size           string      `json:"size"`
	DownloadURL    string      `json:"download_url"`
}

type CacheEntry struct {
	Type         PackageType `json:"type"`
	Slug         string      `json:"slug"`
	Version      string      `json:"version"`
	FilePath     string      `json:"file_path"`
	Size         int64       `json:"size"`
	SHA256       string      `json:"sha256"`
	DownloadedAt string      `json:"downloaded_at"`
}

type Manifest struct {
	SchemaVersion int          `json:"schema_version"`
	Packages      []CacheEntry `json:"packages"`
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
