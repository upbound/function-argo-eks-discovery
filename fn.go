package main

import (
	"context"
	"encoding/base64"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/crossplane-contrib/provider-argocd/apis/cluster/v1alpha1"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"
)

// Function Logger
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(ctx context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) { //nolint:gocyclo // complex
	f.log.Info("Running Function")

	rsp := response.To(req, response.DefaultTTL)

	xr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get observed composite resource from %T", req))
		return rsp, nil
	}

	region, err := xr.Resource.GetString("spec.region")
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot read spec.region field of %s", xr.Resource.GetKind()))
		return rsp, nil
	}

	argoRoleArn, err := xr.Resource.GetString("spec.argoRoleArn")
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot read spec.argoRoleArn field of %s", xr.Resource.GetKind()))
		return rsp, nil
	}

	assumeRoleArn, err := xr.Resource.GetString("spec.assumeRoleArn")
	if err != nil {
		// optional parameter
		assumeRoleArn = ""
	}

	assumeRoleWithWebIdentityArn, err := xr.Resource.GetString("spec.assumeRoleWithWebIdentityArn")
	if err != nil {
		// optional parameter
		assumeRoleWithWebIdentityArn = ""
	}

	tagKey, err := xr.Resource.GetString("spec.search.key")
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot read spec.search.key field of %s", xr.Resource.GetKind()))
		return rsp, nil
	}

	tagValue, err := xr.Resource.GetString("spec.search.value")
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot read spec.search.value field of %s", xr.Resource.GetKind()))
		return rsp, nil
	}

	desired, err := request.GetDesiredComposedResources(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired resources from %T", req))
		return rsp, nil
	}

	// Initialize an AWS session
	session, err := initializeAWSSession(ctx, region, assumeRoleArn, assumeRoleWithWebIdentityArn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load assumed role AWS config")
	}

	// Create an EKS client
	eksClient := eks.NewFromConfig(*session)

	// Initialize the token for pagination
	var nextToken *string

	for {
		// List EKS clusters with the provided token
		input := &eks.ListClustersInput{
			NextToken: nextToken,
		}
		clusters, err := eksClient.ListClusters(ctx, input)
		if err != nil {
			response.Fatal(rsp, errors.Wrapf(err, "failed to list EKS clusters"))
			return rsp, nil
		}

		// Iterate through the clusters and filter by the desired tag
		for _, clusterName := range clusters.Clusters {
			localClusterName := clusterName
			// Describe the cluster to check its tags
			describeInput := &eks.DescribeClusterInput{
				Name: &localClusterName,
			}

			describeOutput, err := eksClient.DescribeCluster(ctx, describeInput)
			if err != nil {
				response.Fatal(rsp, errors.Wrapf(err, "failed to describe cluster %s", clusterName))
				return rsp, nil
			}

			f.log.Info("Found Cluster", "name", describeOutput.Cluster.Name)

			// Check if the cluster has the specified tag
			tagFound := false
			tags := describeOutput.Cluster.Tags
			if tags != nil {
				if value, exists := tags[tagKey]; exists && value == tagValue {
					tagFound = true
				}
			}

			if tagFound {

				f.log.Info("Create ArgoCD Managed Resource for:", "name", describeOutput.Cluster.Name)
				// Assuming describeOutput.Cluster.CertificateAuthority.Data is a base64-encoded string,
				// decode it to []byte
				caData, err := base64.StdEncoding.DecodeString(*describeOutput.Cluster.CertificateAuthority.Data)
				if err != nil {
					response.Fatal(rsp, errors.Wrapf(err, "failed to set caData for cluster %s", clusterName))
				}
				// Add v1beta1 types (including Bucket) to the composed resource scheme.
				// composed.From uses this to automatically set apiVersion and kind.
				_ = v1alpha1.SchemeBuilder.AddToScheme(composed.Scheme)
				// Now you can use the endpoint and Certificate Authority data as needed.
				// For example, you can create a BucketSpec here and add it to desired resources.
				// Create a BucketSpec using cluster details if needed.
				argoCdServerSpec := &v1alpha1.Cluster{
					Spec: v1alpha1.ClusterSpec{
						ForProvider: v1alpha1.ClusterParameters{
							Server: describeOutput.Cluster.Endpoint,
							Name:   &localClusterName,
							Labels: tags,
							Config: v1alpha1.ClusterConfig{
								AWSAuthConfig: &v1alpha1.AWSAuthConfig{
									ClusterName: &localClusterName,
									RoleARN:     &argoRoleArn,
								},
								TLSClientConfig: &v1alpha1.TLSClientConfig{
									CAData: caData,
								},
							},
						},
					},
				}

				cd, err := composed.From(argoCdServerSpec)
				if err != nil {
					response.Fatal(rsp, errors.Wrapf(err, "cannot convert %T to %T", cd, &composed.Unstructured{}))
					return rsp, nil
				}

				f.log.Info("Add ArgoCD Managed Resource for in desired:", "name", describeOutput.Cluster.Name)
				desired[resource.Name(clusterName)] = &resource.DesiredComposed{Resource: cd}
			} else {
				f.log.Info("Cluster not matched:", "name", describeOutput.Cluster.Name)
			}
		}

		// If there are more clusters to retrieve, update the nextToken
		if clusters.NextToken != nil {
			nextToken = clusters.NextToken
		} else {
			break // No more clusters to retrieve
		}
	}

	if err := response.SetDesiredComposedResources(rsp, desired); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources in %T", rsp))
		return rsp, nil
	}

	return rsp, nil
}
