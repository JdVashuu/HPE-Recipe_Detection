# Recipe Detection Helm Chart Structure

This directory contains the Helm chart used to deploy the Recipe Detection Go backend service. It is designed to be dynamically updated by the application's built-in GitOps functionality.

## Directory Structure Overview

```text
helm/recipe-detection-chart/
├── Chart.yaml        # Core chart metadata (name, version, description).
├── values.yaml       # Default configuration values for the deployment.
└── templates/        # Kubernetes manifest templates.
    ├── _helpers.tpl  # Reusable template logic and variable definitions.
    ├── deployment.yaml # K8s Deployment manifest for the Go API.
    ├── service.yaml  # K8s Service to expose the API internally.
    └── configmap.yaml # Dynamically populated ConfigMap for recipe state.
```

## Core Files Delivered

### 1. `Chart.yaml`
Acts as the descriptor for the Helm chart. 
- **Type:** `application`
- **Name:** `recipe-detection`
- **Version Management:** The `version` and `appVersion` fields in this file are automatically bumped by the `GitOpsService` (located in `internal/service/gitops.go`) whenever a new release is processed by the API.

### 2. `values.yaml`
Contains the default configuration for the infrastructure. Key configurations include:
- **`replicaCount: 1`**: Ensures a basic, single-instance deployment by default.
- **`image.tag: "0.0.1"`**: Provides the baseline image tag (which Jenkins overrides during the deployment phase).
- **`resources`**: Establises baseline CPU/Memory requests and limits (`requests: 250m cpu / 256Mi mem`, `limits: 500m cpu / 512Mi mem`) ensuring the application runs efficiently on the cluster without consuming unbounded resources.
- **`recipeData`**: The schema used to map software recipes into the cluster's ConfigMap.

## How it integrates with the Project
This structure forms the backbone of the project's **GitOps pipeline**:
1. The Jenkins CI (`JenkinsFile`) reads `Chart.yaml` to determine what versions to build and deploy.
2. The Go backend's `DeployRelease` API handler clones this chart repository, modifies `values.yaml` dynamically with new recipes, and pushes the changes back—treating this Helm chart as the definitive source of truth for the cluster state.