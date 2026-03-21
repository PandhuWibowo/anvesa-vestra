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

// ── Alibaba Cloud OSS credentials & client ───────────────────────

func ossCredsFromJSON(raw string) (map[string]string, error) {
	var creds map[string]string
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func ossS3Client(ctx context.Context, creds map[string]string) (*s3.Client, error) {
	accessKey := creds["access_key_id"]
	secretKey := creds["secret_access_key"]
	endpoint := creds["endpoint"]
	region := creds["region"]
	if region == "" {
		region = "cn-hangzhou"
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing access_key_id or secret_access_key")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("missing endpoint (e.g. https://oss-cn-hangzhou.aliyuncs.com)")
	}

	provider := awsauth.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(provider),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	}), nil
}

func testOSS(bucket, credentialsJSON string) error {
	creds, err := ossCredsFromJSON(credentialsJSON)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ossS3Client(ctx, creds)
	if err != nil {
		return err
	}
	_, err = client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int32(1),
	})
	return err
}

// Alibaba is the shared S3 provider instance for Alibaba Cloud OSS.
var Alibaba = &S3Provider{
	Name:       "alibaba",
	Table:      "alibaba_connections",
	CredsFunc:  ossCredsFromJSON,
	ClientFunc: ossS3Client,
	TestFunc:   testOSS,
}

// ── Exported handler functions (backward-compatible names) ───────

var (
	ListAlibaba          = Alibaba.ListConnections()
	CreateAlibaba        = Alibaba.CreateConnection()
	AlibabaConnByID      = Alibaba.ConnByID()
	TestAlibaba          = Alibaba.TestConnection()
	BrowseAlibabaBucket  = Alibaba.Browse()
	ListAlibabaObjects   = Alibaba.ListObjects()
	AlibabaDownloadURL   = Alibaba.Download()
	DeleteAlibabaObject  = Alibaba.Delete()
	CopyAlibabaObject    = Alibaba.Copy()
	UploadAlibabaObject  = Alibaba.Upload()
	AlibabaBucketStats   = Alibaba.Stats()
	GetAlibabaMetadata   = Alibaba.GetMetadata()
	UpdateAlibabaMetadata = Alibaba.UpdateMetadata()
	DeletePrefixAlibaba  = Alibaba.DeletePrefix()
)
