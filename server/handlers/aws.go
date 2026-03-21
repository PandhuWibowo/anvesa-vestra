package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsauth "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ── AWS S3 credentials & client ──────────────────────────────────

func awsCredsFromJSON(raw string) (map[string]string, error) {
	var creds map[string]string
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func awsS3Client(ctx context.Context, creds map[string]string) (*s3.Client, error) {
	accessKey := creds["access_key_id"]
	secretKey := creds["secret_access_key"]
	token := creds["session_token"]
	region := creds["region"]
	if region == "" {
		region = "us-east-1"
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing access_key_id or secret_access_key")
	}

	provider := awsauth.NewStaticCredentialsProvider(accessKey, secretKey, token)
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(provider),
	)
	if err != nil {
		return nil, err
	}

	opts := []func(*s3.Options){}
	if ep := creds["endpoint"]; ep != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(ep)
			o.UsePathStyle = true
		})
	}
	return s3.NewFromConfig(cfg, opts...), nil
}

func testS3(bucket, credentialsJSON string) error {
	creds, err := awsCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := awsS3Client(ctx, creds)
	if err != nil {
		return err
	}
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	return err
}

// AWS is the shared S3 provider instance for Amazon S3 / R2 / MinIO.
var AWS = &S3Provider{
	Name:       "aws",
	Table:      "aws_connections",
	CredsFunc:  awsCredsFromJSON,
	ClientFunc: awsS3Client,
	TestFunc:   testS3,
}

// ── Exported handler functions (backward-compatible names) ───────

var (
	ListAWS          = AWS.ListConnections()
	CreateAWS        = AWS.CreateConnection()
	AWSConnByID      = AWS.ConnByID()
	TestAWS          = AWS.TestConnection()
	BrowseAWSBucket  = AWS.Browse()
	ListAWSObjects   = AWS.ListObjects()
	AWSDownloadURL   = AWS.Download()
	DeleteAWSObject  = AWS.Delete()
	CopyAWSObject    = AWS.Copy()
	UploadAWSObject  = AWS.Upload()
	AWSBucketStats   = AWS.Stats()
	GetAWSMetadata   = AWS.GetMetadata()
	UpdateAWSMetadata = AWS.UpdateMetadata()
	DeletePrefixAWS  = AWS.DeletePrefix()
)
