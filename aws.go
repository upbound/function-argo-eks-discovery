package main

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
)

const webIdentityTokenFileDefaultPath = "/var/run/secrets/upbound.io/function/token"

func getWebidentityTokenFilePath() string {
	if path := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"); path != "" {
		return path
	}
	return webIdentityTokenFileDefaultPath
}

func initializeAWSSession(ctx context.Context, region, assumeRoleArn, assumeRoleWithWebIdentityArn string) (*aws.Config, error) {
	cfg, _ := config.LoadDefaultConfig(ctx)
	cfg.Region = region
	stsclient := sts.NewFromConfig(cfg)
	session, _ := config.LoadDefaultConfig(ctx)

	var err error
	if assumeRoleArn != "" {
		session, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(
				stscreds.NewAssumeRoleProvider(
					stsclient,
					assumeRoleArn,
				)),
			),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load assumed role AWS config")
		}
	}

	if assumeRoleWithWebIdentityArn != "" {
		session, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(
				stscreds.NewWebIdentityRoleProvider(
					stsclient,
					assumeRoleWithWebIdentityArn,
					stscreds.IdentityTokenFile(getWebidentityTokenFilePath()),
				)),
			),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load assumed with webidentity role AWS config")
		}
	}

	return &session, nil
}
