package sitemap

import (
	"context"
	"errors"
	"fmt"
	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
	m "github.com/minio/minio-go/v7"
	"github.com/telq-org/sitemap-builder/pkg/config"
	"github.com/telq-org/sitemap-builder/pkg/logger"
	"github.com/telq-org/sitemap-builder/pkg/minio"
	"github.com/telq-org/sitemap-builder/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type document struct {
	ID primitive.ObjectID `bson:"_id"`
}

const outFolderName = "out"

func Build() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	defer func() {
		wd, err := os.Getwd()
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return
		}

		err = os.RemoveAll(path.Join(wd, outFolderName))
		if err != nil {
			logger.Log.Error().Err(err).Send()
		}
	}()

	totalProcessed := 0
	go func(c context.Context) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.Done():
				return
			case <-ticker.C:
				logger.Log.Debug().Int("totalProcessed", totalProcessed).Send()
			}
		}
	}(ctx)

	const https = "https://"

	sm := stm.NewSitemap(0)
	sm.SetDefaultHost(https + "telq.org")
	sm.SetSitemapsHost(https + "sitemap.telq.org")
	sm.SetSitemapsPath("")
	sm.SetPublicPath(outFolderName)
	sm.SetAdapter(stm.NewFileAdapter())
	sm.SetVerbose(false)
	sm.Create()

	cur, err := mongo.Threads.Find(ctx, bson.M{})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	for cur.Next(ctx) {
		var doc document
		err := cur.Decode(&doc)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}

		sm.Add(stm.URL{{"loc", strings.Join([]string{
			"question",
			doc.ID.Hex(),
		}, "/")}})
		totalProcessed += 1
	}

	sm.Finalize()

	return uploadToS3(ctx)
}

func truncateS3Bucket(ctx context.Context) error {
	for obj := range minio.Client.ListObjects(ctx, config.Env.S3.BucketSitemap, m.ListObjectsOptions{}) {
		err := minio.Client.RemoveObject(ctx, config.Env.S3.BucketSitemap, obj.Key, m.RemoveObjectOptions{})
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}
	}
	return nil
}

func uploadToS3(ctx context.Context) error {
	err := truncateS3Bucket(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	for i := 0; true; i += 1 {
		filename := ""
		if i == 0 {
			filename = "sitemap.xml.gz"
		} else {
			filename = fmt.Sprintf("sitemap%s.xml.gz", strconv.Itoa(i))
		}
		filepath := path.Join(wd, outFolderName, filename)

		_, err := minio.Client.FPutObject(ctx, config.Env.S3.BucketSitemap, filename, filepath, m.PutObjectOptions{})
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}

			logger.Log.Error().Err(err).Send()
			return err
		}
	}

	return nil
}
