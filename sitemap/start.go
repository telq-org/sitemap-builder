package sitemap

import (
	"context"
	"errors"
	"fmt"
	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"
	m "github.com/minio/minio-go/v7"
	"github.com/nnqq/scr-sitemap-builder/config"
	"github.com/nnqq/scr-sitemap-builder/logger"
	"github.com/nnqq/scr-sitemap-builder/minio"
	"github.com/nnqq/scr-sitemap-builder/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type document struct {
	Slug string `bson:"s"`
}

const outFolderName = "out"

func Build() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

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

	const (
		https    = "https://"
		download = "download"
	)

	sm := stm.NewSitemap(0)
	sm.SetDefaultHost(https + config.Env.Host.URL)
	sm.SetSitemapsHost(https + "sitemap." + config.Env.Host.URL)
	sm.SetSitemapsPath("")
	sm.SetPublicPath(outFolderName)
	sm.SetAdapter(stm.NewFileAdapter())
	sm.SetVerbose(false)
	sm.Create()

	curCity, err := mongo.Cities.Find(ctx, bson.D{})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	sm.Add(stm.URL{{"loc", "about"}})
	sm.Add(stm.URL{{"loc", "cities"}})
	sm.Add(stm.URL{{"loc", "categories"}})
	sm.Add(stm.URL{{"loc", "plans"}})
	sm.Add(stm.URL{{"loc", "all/all"}})
	sm.Add(stm.URL{{"loc", "all/all/download"}})
	totalProcessed += 4
	for curCity.Next(ctx) {
		var city document
		err := curCity.Decode(&city)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}

		if city.Slug == "" {
			continue
		}

		sm.Add(stm.URL{{"loc", strings.Join([]string{
			city.Slug,
			"all",
		}, "/")}})
		sm.Add(stm.URL{{"loc", strings.Join([]string{
			city.Slug,
			"all",
			download,
		}, "/")}})
		totalProcessed += 2

		curCategory, err := mongo.Categories.Find(ctx, bson.D{})
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}

		for curCategory.Next(ctx) {
			var category document
			err := curCategory.Decode(&category)
			if err != nil {
				logger.Log.Error().Err(err).Send()
				return err
			}

			if category.Slug == "" {
				continue
			}

			sm.Add(stm.URL{{"loc", strings.Join([]string{
				city.Slug,
				category.Slug,
			}, "/")}})
			sm.Add(stm.URL{{"loc", strings.Join([]string{
				city.Slug,
				category.Slug,
				download,
			}, "/")}})
			totalProcessed += 2
		}

		err = curCategory.Close(ctx)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}
	}

	curCategory, err := mongo.Categories.Find(ctx, bson.D{})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	for curCategory.Next(ctx) {
		var category document
		err := curCategory.Decode(&category)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}

		if category.Slug == "" {
			continue
		}

		sm.Add(stm.URL{{"loc", strings.Join([]string{
			"all",
			category.Slug,
		}, "/")}})
		sm.Add(stm.URL{{"loc", strings.Join([]string{
			"all",
			category.Slug,
			download,
		}, "/")}})
		totalProcessed += 2
	}

	curCompany, err := mongo.Companies.Find(ctx, bson.M{
		"h": false,
	})
	if err != nil {
		logger.Log.Error().Err(err).Send()
		return err
	}

	for curCompany.Next(ctx) {
		var company document
		err := curCompany.Decode(&company)
		if err != nil {
			logger.Log.Error().Err(err).Send()
			return err
		}

		if company.Slug == "" {
			continue
		}

		sm.Add(stm.URL{{"loc", strings.Join([]string{
			"company",
			company.Slug,
		}, "/")}})
		totalProcessed += 1
	}

	sm.Finalize()

	return uploadToS3(ctx)
}

func truncateS3Bucket(ctx context.Context) error {
	for obj := range minio.Client.ListObjects(ctx, config.Env.S3.SitemapBucketName, m.ListObjectsOptions{}) {
		err := minio.Client.RemoveObject(ctx, config.Env.S3.SitemapBucketName, obj.Key, m.RemoveObjectOptions{})
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

		_, err := minio.Client.FPutObject(ctx, config.Env.S3.SitemapBucketName, filename, filepath, m.PutObjectOptions{})
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
