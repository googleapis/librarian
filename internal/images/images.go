// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package images provides operations around docker images.
package images

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	artifactregistry "cloud.google.com/go/artifactregistry/apiv1"
	artifactregistrypb "cloud.google.com/go/artifactregistry/apiv1/artifactregistrypb"
)

// ImageRegistryClient is an abstration of a Docker registry client.
type ImageRegistryClient interface {
	// FindLatest returns the latest docker image given a current image.
	FindLatest(ctx context.Context, image *Image) (string, error)
	// Close cleans up any open resources.
	Close() error
}

// ArtifactRegistryClient is the implementation of ImageRegistryClient
// to interact with Artifact Registry.
type ArtifactRegistryClient struct {
	client *artifactregistry.Client
}

// Image is a data structure for parsing Docker image parameters.
type Image struct {
	Name       string
	Tag        string
	SHA        string
	Location   string
	Project    string
	Repository string
}

// BaseName returns the image name without a pinned SHA or tag.
func (i *Image) BaseName() string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s", i.Location, i.Project, i.Repository, i.Name)
}

// String returns the image name with pinned SHA or tag.
func (i *Image) String() string {
	var b strings.Builder
	b.WriteString(i.BaseName())
	if i.SHA != "" {
		b.WriteString("@")
		b.WriteString(i.SHA)
	} else if i.Tag != "" {
		b.WriteString(":")
		b.WriteString(i.Tag)
	}
	return b.String()
}

// NewArtifactRegistryClient creates a new ArtifactRegistryClient.
func NewArtifactRegistryClient(ctx context.Context) (*ArtifactRegistryClient, error) {
	client, err := artifactregistry.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &ArtifactRegistryClient{
		client: client,
	}, nil
}

// Close cleans up any open resources.
func (c *ArtifactRegistryClient) Close() error {
	return c.client.Close()
}

// FindLatest returns the latest docker image given a current image.
func (c *ArtifactRegistryClient) FindLatest(ctx context.Context, image *Image) (string, error) {
	if c.client == nil {
		return "", fmt.Errorf("no client configured")
	}

	it := c.client.ListVersions(ctx, &artifactregistrypb.ListVersionsRequest{
		Parent:  fmt.Sprintf("projects/%s/locations/%s/repositories/%s/packages/%s", image.Project, image.Location, image.Repository, image.Name),
		View:    artifactregistrypb.VersionView_FULL,
		OrderBy: "create_time DESC",
	})
	version, err := it.Next()
	if err != nil {
		return "", err
	}
	slog.Info("Found packages version", "version", version.GetName())

	// latest SHA is found as the "subjectDigest" metadata field
	latestSha := ""
	for key, field := range version.GetMetadata().GetFields() {
		if key == "subjectDigest" {
			slog.Info("Found SHA", "sha", field.GetStringValue())
			latestSha = field.GetStringValue()
			break
		}
	}

	if latestSha == "" {
		return "", fmt.Errorf("failed to find updated SHA for latest version: %s", version.GetName())
	}

	newImage := &Image{
		Name:       image.Name,
		Location:   image.Location,
		Repository: image.Repository,
		Project:    image.Project,
		SHA:        latestSha,
	}
	return newImage.String(), nil
}

// FindLatestImage returns the latest created image matching the provided image name.
func FindLatestImage(ctx context.Context, client ImageRegistryClient, currentImage string) (string, error) {
	image, err := parseImage(currentImage)
	if err != nil {
		return "", err
	}

	latestImage, err := client.FindLatest(ctx, image)
	if err != nil {
		return "", err
	}
	return latestImage, nil
}

func parseImage(pinnedImage string) (*Image, error) {
	parsedImage := &Image{}
	baseName := ""

	atParts := strings.Split(pinnedImage, "@")
	colonParts := strings.Split(pinnedImage, ":")
	if len(atParts) == 2 {
		baseName = atParts[0]
		parsedImage.SHA = atParts[1]
	} else if len(colonParts) == 2 {
		baseName = colonParts[0]
		parsedImage.Tag = colonParts[1]
	}

	if baseName == "" {
		slog.Info("image does not appear to be pinned")
		baseName = pinnedImage
	}

	parts := strings.Split(baseName, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected image format, expected an AR formatted image: %s", baseName)
	}

	host := parts[0]
	if strings.HasSuffix(host, "-docker.pkg.dev") {
		parsedImage.Location = strings.TrimSuffix(host, "-docker.pkg.dev")
	} else {
		return nil, fmt.Errorf("expected AR formatted host with -docker.pkg.dev suffix: %s", host)
	}

	parsedImage.Project = parts[1]
	parsedImage.Repository = parts[2]
	parsedImage.Name = parts[3]

	return parsedImage, nil
}
