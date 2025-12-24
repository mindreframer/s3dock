package internal

// CommandResult is the generic wrapper for all command results in JSON mode
type CommandResult struct {
	Success bool        `json:"success"`
	Command string      `json:"command"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BuildResult contains the result of a build command
type BuildResult struct {
	ImageTag string `json:"image_tag"`
	AppName  string `json:"app_name"`
	GitHash  string `json:"git_hash"`
	GitTime  string `json:"git_time"`
}

// PushResult contains the result of a push command
type PushResult struct {
	ImageRef string `json:"image_ref"`
	S3Key    string `json:"s3_key"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	Skipped  bool   `json:"skipped"`
	Archived bool   `json:"archived"`
}

// TagResult contains the result of a tag command
type TagResult struct {
	ImageRef string `json:"image_ref"`
	Version  string `json:"version"`
	S3Key    string `json:"s3_key"`
}

// PromoteResult contains the result of a promote command
type PromoteResult struct {
	Source      string `json:"source"`
	Environment string `json:"environment"`
	SourceType  string `json:"source_type"` // "image" or "tag"
	ImageRef    string `json:"image_ref"`
	Skipped     bool   `json:"skipped"`
}

// PullResult contains the result of a pull command
type PullResult struct {
	ImageRef   string `json:"image_ref"`
	Source     string `json:"source"`
	SourceType string `json:"source_type"` // "environment" or "tag"
	Skipped    bool   `json:"skipped"`
}

// CurrentResult contains the result of a current command
type CurrentResult struct {
	AppName     string `json:"app_name"`
	Environment string `json:"environment"`
	ImageRef    string `json:"image_ref"`
}

// ListAppsResult contains the result of a list apps command
type ListAppsResult struct {
	Apps []string `json:"apps"`
}

// ListImagesResult contains the result of a list images command
type ListImagesResult struct {
	AppName   string          `json:"app_name"`
	YearMonth string          `json:"year_month,omitempty"`
	Images    []ImageInfoJSON `json:"images"`
}

// ImageInfoJSON is the JSON-serializable version of ImageInfo
type ImageInfoJSON struct {
	AppName   string `json:"app_name"`
	Tag       string `json:"tag"`
	S3Path    string `json:"s3_path"`
	YearMonth string `json:"year_month"`
}

// ListTagsResult contains the result of a list tags command
type ListTagsResult struct {
	AppName string        `json:"app_name"`
	Tags    []TagInfoJSON `json:"tags"`
}

// TagInfoJSON is the JSON-serializable version of TagInfo
type TagInfoJSON struct {
	AppName     string `json:"app_name"`
	Version     string `json:"version"`
	TargetImage string `json:"target_image"`
	S3Path      string `json:"s3_path"`
}

// ListEnvsResult contains the result of a list envs command
type ListEnvsResult struct {
	AppName      string        `json:"app_name"`
	Environments []EnvInfoJSON `json:"environments"`
}

// EnvInfoJSON is the JSON-serializable version of EnvInfo
type EnvInfoJSON struct {
	AppName     string `json:"app_name"`
	Environment string `json:"environment"`
	TargetType  string `json:"target_type"` // "image" or "tag"
	TargetPath  string `json:"target_path"`
	SourceTag   string `json:"source_tag,omitempty"`
	SourceImage string `json:"source_image"`
}

// ListTagForResult contains the result of a list tag-for command
type ListTagForResult struct {
	AppName     string `json:"app_name"`
	Environment string `json:"environment"`
	Tag         string `json:"tag"`
	Direct      bool   `json:"direct"` // true if promoted directly from image (no tag)
}

// VersionResult contains the result of a version command
type VersionResult struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// ConfigShowResult contains the result of a config show command
type ConfigShowResult struct {
	Profile   string `json:"profile"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	Endpoint  string `json:"endpoint,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
}

// ConfigListResult contains the result of a config list command
type ConfigListResult struct {
	Profiles       []string `json:"profiles"`
	DefaultProfile string   `json:"default_profile"`
}

// ToImageInfoJSON converts ImageInfo to ImageInfoJSON
func (i ImageInfo) ToJSON() ImageInfoJSON {
	return ImageInfoJSON{
		AppName:   i.AppName,
		Tag:       i.Tag,
		S3Path:    i.S3Path,
		YearMonth: i.YearMonth,
	}
}

// ToTagInfoJSON converts TagInfo to TagInfoJSON
func (t TagInfo) ToJSON() TagInfoJSON {
	return TagInfoJSON{
		AppName:     t.AppName,
		Version:     t.Version,
		TargetImage: t.TargetImage,
		S3Path:      t.S3Path,
	}
}

// ToEnvInfoJSON converts EnvInfo to EnvInfoJSON
func (e EnvInfo) ToJSON() EnvInfoJSON {
	return EnvInfoJSON{
		AppName:     e.AppName,
		Environment: e.Environment,
		TargetType:  string(e.TargetType),
		TargetPath:  e.TargetPath,
		SourceTag:   e.SourceTag,
		SourceImage: e.SourceImage,
	}
}
