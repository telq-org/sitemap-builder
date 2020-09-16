package config

import (
	"github.com/kelseyhightower/envconfig"
)

type c struct {
	S3       s3
	MongoDB  mongodb
	Host     host
	LogLevel string `envconfig:"LOGLEVEL"`
}

type s3 struct {
	SitemapBucketName string `envconfig:"S3_SITEMAPBUCKETNAME"`
	Endpoint          string `envconfig:"S3_ENDPOINT"`
	AccessKeyID       string `envconfig:"S3_ACCESSKEYID"`
	SecretAccessKey   string `envconfig:"S3_SECRETACCESSKEY"`
	Secure            string `envconfig:"S3_SECURE"`
	Region            string `envconfig:"S3_REGION"`
}

type mongodb struct {
	URL string `envconfig:"MONGODB_URL"`
}

type host struct {
	URL string `envconfig:"HOST_URL"`
}

var Env c

func init() {
	envconfig.MustProcess("", &Env)
}
