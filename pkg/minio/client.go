package minio

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/telq-org/sitemap-builder/pkg/config"
	"github.com/telq-org/sitemap-builder/pkg/logger"
	"strconv"
	"time"
)

var Client *minio.Client

func init() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secure, err := strconv.ParseBool(config.Env.S3.Secure)
	logger.Must(err)

	cl, err := minio.New(config.Env.S3.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			config.Env.S3.AccessKeyID,
			config.Env.S3.SecretAccessKey,
			"",
		),
		Secure: secure,
	})
	logger.Must(err)

	// ping
	_, err = cl.ListBuckets(ctx)
	logger.Must(err)

	err = cl.MakeBucket(ctx, config.Env.S3.BucketSitemap, minio.MakeBucketOptions{
		Region: config.Env.S3.Region,
	})
	if err != nil {
		// ok, seems bucket exists
		logger.Log.Debug().Err(err).Send()
	} else {
		logger.Log.Debug().Str("bucketName", config.Env.S3.BucketSitemap).Msg("bucket created")
	}

	//err = cl.SetBucketPolicy(ctx, config.Env.S3.BucketSitemap, fmt.Sprintf(`{
	//	"Version": "2012-10-17",
	//	"Statement": [{
	//		"Sid": "PublicRead",
	//		"Effect": "Allow",
	//		"Principal": "*",
	//		"Action": ["s3:GetObject"],
	//		"Resource": ["arn:aws:s3:::%s/*"]
	//	}]
	//}`, config.Env.S3.BucketSitemap))
	//if err != nil && err.Error() != "200 OK" {
	//	logger.Log.Panic().Err(err).Send()
	//}

	Client = cl
}
