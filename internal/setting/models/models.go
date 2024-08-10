package models

type UploadProvider struct {
	UploadProvider             string `json:"upload.provider"`
	UploadFilesystemUploadPath string `json:"upload.localfs.upload_path"`
	UploadS3URL                string `json:"upload.s3.url"`
	UploadS3AwsAccessKeyID     string `json:"upload.s3.access_key"`
	UploadS3AwsSecretAccessKey string `json:"upload.s3.access_secret"`
	UploadS3AwsDefaultRegion   string `json:"upload.s3.region"`
	UploadS3Bucket             string `json:"upload.s3.bucket"`
	UploadS3BucketPath         string `json:"upload.s3.bucket_path"`
	UploadS3BucketType         string `json:"upload.s3.bucket_type"`
	UploadS3Expiry             string `json:"upload.s3.upload_expiry"`
}

type General struct {
	SiteName                    string   `json:"app.site_name"`
	Lang                        string   `json:"app.lang"`
	MaxFileUploadSize           int      `json:"app.max_file_upload_size"`
	FaviconURL                  string   `json:"app.favicon_url"`
	RootURL                     string   `json:"app.root_url"`
	AllowedFileUploadExtensions []string `json:"app.allowed_file_upload_extensions"`
}

type Settings struct {
	UploadProvider
	General
}
