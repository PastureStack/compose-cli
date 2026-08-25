package platformapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/PastureStack/compose-cli/project"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/sirupsen/logrus"
)

type S3Uploader struct {
}

func (s *S3Uploader) Name() string {
	return "S3"
}

func (s *S3Uploader) Upload(p *project.Project, name string, reader io.ReadSeeker, hash string) (string, string, error) {
	bucketName := fmt.Sprintf("%s-%s", p.Name, workspaceHash())
	objectKey := fmt.Sprintf("%s-%s", name, hash[:12])

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	config, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithDefaultRegion("us-east-1"))
	if err != nil {
		return "", "", err
	}
	svc := s3.NewFromConfig(config)

	if err := getOrCreateBucket(ctx, svc, config.Region, bucketName); err != nil {
		return "", "", err
	}

	if err := putFile(ctx, svc, bucketName, objectKey, reader); err != nil {
		return "", "", err
	}

	request, err := s3.NewPresignClient(svc).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(options *s3.PresignOptions) {
		options.Expires = 7 * 24 * time.Hour
	})
	if err != nil {
		return "", "", err
	}
	return objectKey, request.URL, nil
}

func putFile(ctx context.Context, svc *s3.Client, bucket, object string, reader io.ReadSeeker) error {
	_, err := svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(object),
		Body:        reader,
		ContentType: aws.String("application/tar"),
	})

	return err
}

func getOrCreateBucket(ctx context.Context, svc *s3.Client, region, bucketName string) error {
	_, err := svc.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	var responseError *smithyhttp.ResponseError
	if errors.As(err, &responseError) && responseError.HTTPStatusCode() == 404 {
		logrus.Infof("Creating bucket %s", bucketName)
		input := &s3.CreateBucketInput{Bucket: aws.String(bucketName)}
		if region != "" && region != "us-east-1" {
			input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
				LocationConstraint: types.BucketLocationConstraint(region),
			}
		}
		_, err = svc.CreateBucket(ctx, input)
	}

	return err
}

func workspaceHash() string {
	sha := sha256.New()

	wd, err := os.Getwd()
	if err == nil {
		sha.Write([]byte(wd))
	}

	return hex.EncodeToString(sha.Sum(nil))[:12]
}
