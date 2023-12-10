# function-argo-eks-discovery
This GitHub repository contains Go code for a Crossplane function that facilitates interaction with Amazon Elastic Kubernetes Service (EKS) clusters. The code is designed to run as a part of Crossplane, allowing you to automate tasks related to EKS clusters within your cloud-native infrastructure.

# Overview
The primary purpose of this code is to interact with EKS clusters, fetch cluster information, filter clusters based on specific tags, and create ArgoCD Server managed resources that meet the specified criteria.

# The code performs the following actions:

1. Retrieves parameters and information from the input request.
2. Initializes an AWS session and an EKS client using the AWS SDK for Go.
3. Lists all EKS clusters in the specified AWS region.
4. Filters clusters based on user-defined tags.
5. For each matching cluster, it decodes the certificate authority data, endpoint and generates a ArgoCD Server managed resource.
This ArgoCD Server managed resource is added to the desired resources to be returned as part of the response.

# Prerequisites
Before using this code, make sure you have the following prerequisites:

- An existing Crossplane installation.
- Appropriate AWS IAM permissions to access EKS clusters and read their tags.

# Contributing
We welcome contributions to this repository. If you have improvements, bug fixes, or additional features to add, please submit a pull request, following our guidelines for contribution.
