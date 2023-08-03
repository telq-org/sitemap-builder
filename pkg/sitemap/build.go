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
	mongod "go.mongodb.org/mongo-driver/mongo"
	"os"
	"path"
	"strings"
	"time"
)

type document struct {
	ID        primitive.ObjectID `bson:"_id"`
	UpdatedAt time.Time          `bson:"ua"`
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

	sm := stm.NewSitemap(1)
	sm.SetFilename("s")
	sm.SetCompress(false)
	sm.SetDefaultHost(https + "telq.org")
	sm.SetSitemapsHost(https + "telq.org")
	sm.SetSitemapsPath("sitemap")
	sm.SetPublicPath(outFolderName)
	sm.SetAdapter(stm.NewFileAdapter())
	sm.SetVerbose(false)
	sm.Create()

	sm.Add(stm.URL{{
		"loc",
		"",
	}, {
		"priority",
		"1.0",
	}, {
		"changefreq",
		"daily",
	}, {
		"lastmod",
		time.Now().UTC().Format(time.RFC3339),
	}})
	sm.Add(stm.URL{{
		"loc",
		"communities",
	}, {
		"priority",
		"1.0",
	}, {
		"changefreq",
		"daily",
	}, {
		"lastmod",
		time.Now().UTC().Format(time.RFC3339),
	}})

	err := iterate(
		ctx,
		mongo.Threads,
		bson.M{},
		sm,
		"question",
		"0.7",
		"weekly",
		&totalProcessed,
	)
	if err != nil {
		return fmt.Errorf("iterate question: %w", err)
	}

	err = iterate(
		ctx,
		mongo.Users,
		bson.M{},
		sm,
		"user",
		"0.8",
		"daily",
		&totalProcessed,
	)
	if err != nil {
		return fmt.Errorf("iterate user: %w", err)
	}

	err = iterate(
		ctx,
		mongo.Tags,
		bson.M{},
		sm,
		"tag",
		"0.8",
		"daily",
		&totalProcessed,
	)
	if err != nil {
		return fmt.Errorf("iterate tag: %w", err)
	}

	err = iterate(
		ctx,
		mongo.Communities,
		bson.M{},
		sm,
		"community",
		"0.9",
		"daily",
		&totalProcessed,
	)
	if err != nil {
		return fmt.Errorf("iterate community: %w", err)
	}

	sm.Finalize()

	return uploadToS3(ctx)
}

func iterate(
	ctx context.Context,
	coll *mongod.Collection,
	query interface{},
	sm *stm.Sitemap,
	page,
	priority,
	changefreq string,
	counter *int,
) error {
	cur, err := coll.Find(ctx, query)
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return fmt.Errorf("coll.Find: %w", err)
	}

	for cur.Next(ctx) {
		var doc document
		e := cur.Decode(&doc)
		if e != nil {
			logger.Log.Error().Err(e).Send()
			return fmt.Errorf("cur.Decode: %w", e)
		}

		lastmod := doc.UpdatedAt.Format(time.RFC3339)
		if doc.UpdatedAt.IsZero() {
			const defaultTime = "2023-08-03T20:00:00Z"
			t, er := time.Parse(time.RFC3339, defaultTime)
			if er != nil {
				logger.Log.Error().Err(er).Send()
				return fmt.Errorf("time.Parse: %w", er)
			}
			lastmod = t.Format(time.RFC3339)
		}

		sm.Add(stm.URL{{
			"loc",
			strings.Join([]string{
				page,
				doc.ID.Hex(),
			}, "/"),
		}, {
			"priority",
			priority,
		}, {
			"changefreq",
			changefreq,
		}, {
			"lastmod",
			lastmod,
		}})
		*counter += 1
	}

	return nil
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
			filename = "s.xml"
		} else {
			filename = fmt.Sprintf("s%d.xml", i)
		}
		filepath := path.Join(wd, outFolderName, "sitemap", filename)

		_, e := minio.Client.FPutObject(ctx, config.Env.S3.BucketSitemap, filename, filepath, m.PutObjectOptions{})
		if e != nil {
			if errors.Is(e, os.ErrNotExist) {
				return nil
			}

			logger.Log.Error().Err(e).Send()
			return e
		}
	}

	return nil
}
