package models

type UploadProvider struct {
	UploadProvider             string `json:"upload.provider,omitempty"`
	UploadFilesystemUploadPath string `json:"upload.localfs.upload_path,omitempty"`
	UploadS3URL                string `json:"upload.s3.url,omitempty"`
	UploadS3AwsAccessKeyID     string `json:"upload.s3.access_key,omitempty"`
	UploadS3AwsSecretAccessKey string `json:"upload.s3.access_secret,omitempty"`
	UploadS3AwsDefaultRegion   string `json:"upload.s3.region,omitempty"`
	UploadS3Bucket             string `json:"upload.s3.bucket,omitempty"`
	UploadS3BucketPath         string `json:"upload.s3.bucket_path,omitempty"`
	UploadS3BucketType         string `json:"upload.s3.bucket_type,omitempty"`
	UploadS3Expiry             string `json:"upload.s3.upload_expiry,omitempty"`
}

type General struct {
	SiteName                    string   `json:"app.site_name,omitempty"`
	Lang                        string   `json:"app.lang,omitempty"`
	MaxFileUploadSize           int      `json:"app.max_file_upload_size,omitempty"`
	FaviconURL                  string   `json:"app.favicon_url,omitempty"`
	RootURL                     string   `json:"app.root_url,omitempty"`
	AllowedFileUploadExtensions []string `json:"app.allowed_file_upload_extensions,omitempty"`
}

type Settings struct {
	UploadProvider
	General
}
