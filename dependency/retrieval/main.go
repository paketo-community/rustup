// Copyright 2018-2026 the original author or authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/libdependency/buildpack_config"
	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/upstream"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
)

const (
	purlName = "rustup"

	org  = "rust-lang"
	repo = "rustup"
)

var dependencies = []rustupDependency{
	{id: "rustup-init-gnu", name: "Rustup (GNU libc)", libc: "gnu"},
	{id: "rustup-init-musl", name: "Rustup (musl libc)", libc: "musl"},
}

type rustupDependency struct {
	id   string
	name string
	libc string
}

type tag struct {
	Name string `json:"name"`
}

func main() {
	buildpackTomlPath, output := parseArgs()

	config, err := buildpack_config.ParseBuildpackToml(buildpackTomlPath)
	if err != nil {
		panic(err)
	}

	targets := config.Targets
	if len(targets) == 0 {
		targets = []cargo.ConfigTarget{{OS: "linux", Arch: "amd64"}}
	}

	var (
		versionsOnce sync.Once
		versions     versionology.VersionFetcherArray
		versionsErr  error
	)
	getAllVersions := func() (versionology.VersionFetcherArray, error) {
		versionsOnce.Do(func() {
			versions, versionsErr = fetchTags(org, repo)
		})
		return versions, versionsErr
	}

	var metadata []versionology.Dependency
	for _, dependency := range dependencies {
		newVersions, err := retrieve.GetNewVersionsForId(dependency.id, config, getAllVersions)
		if err != nil {
			panic(err)
		}

		for _, target := range targets {
			platform := retrieve.Platform{OS: target.OS, Arch: target.Arch}
			metadata = append(metadata, retrieve.GenerateAllMetadataWithPlatform(newVersions, generateMetadata(dependency), platform)...)
		}
	}

	contents, err := json.Marshal(metadata)
	if err != nil {
		panic(fmt.Errorf("unable to marshal metadata\n%w", err))
	}

	if err := os.WriteFile(output, contents, 0644); err != nil {
		panic(fmt.Errorf("cannot write to %s: %w", output, err))
	}
	fmt.Printf("Wrote metadata to %s\n", output)
}

func parseArgs() (string, string) {
	var buildpackTomlPath, output string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&buildpackTomlPath, "buildpack-toml-path", "", "full path to the buildpack.toml file")
	flags.StringVar(&output, "output", "", "filename for the output JSON metadata")
	if err := flags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	if buildpackTomlPath == "" {
		panic("buildpack-toml-path is required")
	}
	if output == "" {
		panic("output is required")
	}

	return buildpackTomlPath, output
}

func generateMetadata(dependency rustupDependency) retrieve.GenerateMetadataWithPlatformFunc {
	return func(versionFetcher versionology.VersionFetcher, platform retrieve.Platform) ([]versionology.Dependency, error) {
		rustArch := rustArch(platform.Arch)
		if rustArch == "" {
			fmt.Printf("Skipping %s/%s: unsupported architecture\n", platform.OS, platform.Arch)
			return nil, nil
		}

		versionString := versionFetcher.Version().String()

		source := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", org, repo, versionString)
		sourceChecksum, err := upstream.GetSHA256OfRemoteFile(source)
		if err != nil {
			return nil, fmt.Errorf("unable to checksum %s\n%w", source, err)
		}

		uri := fmt.Sprintf("https://static.rust-lang.org/rustup/archive/%s/%s-unknown-linux-%s/rustup-init", versionString, rustArch, dependency.libc)
		checksum, err := upstream.GetSHA256OfRemoteFile(uri)
		if err != nil {
			return nil, fmt.Errorf("unable to checksum %s\n%w", uri, err)
		}

		configMetadataDependency := cargo.ConfigMetadataDependency{
			Arch:     platform.Arch,
			Checksum: fmt.Sprintf("sha256:%s", checksum),
			CPE:      fmt.Sprintf("cpe:2.3:a:rust:rustup:%s:*:*:*:*:*:*:*", versionString),
			ID:       dependency.id,
			Licenses: []interface{}{
				map[string]string{
					"type": "Apache-2.0",
					"uri":  "https://github.com/rust-lang/rustup/blob/master/LICENSE-APACHE",
				},
				map[string]string{
					"type": "MIT",
					"uri":  "https://github.com/rust-lang/rustup/blob/master/LICENSE-MIT",
				},
			},
			Name:           dependency.name,
			OS:             platform.OS,
			PURL:           purlWithArch(retrieve.GeneratePURL(purlName, versionString, checksum, uri), platform.Arch),
			SHA256:         checksum,
			Source:         source,
			SourceChecksum: fmt.Sprintf("sha256:%s", sourceChecksum),
			Stacks:         []string{"*"},
			URI:            uri,
			Version:        versionString,
		}

		d, err := versionology.NewDependency(configMetadataDependency, fmt.Sprintf("%s-%s", platform.OS, platform.Arch))
		if err != nil {
			return nil, err
		}

		return []versionology.Dependency{d}, nil
	}
}

// purlWithArch adds the arch qualifier to a PURL.
// libdependency's GeneratePURL does not set it.
func purlWithArch(purl string, arch string) string {
	if base, qualifiers, found := strings.Cut(purl, "?"); found {
		return fmt.Sprintf("%s?arch=%s&%s", base, arch, qualifiers)
	}
	return fmt.Sprintf("%s?arch=%s", purl, arch)
}

func rustArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return ""
	}
}

func fetchTags(owner, repo string) (versionology.VersionFetcherArray, error) {
	seen := map[string]struct{}{}
	var versions versionology.VersionFetcherArray

	for page := 1; ; page++ {
		tags, err := fetchTagsPage(owner, repo, page)
		if err != nil {
			return nil, err
		}

		if len(tags) == 0 {
			break
		}

		for _, t := range tags {
			value := strings.TrimPrefix(t.Name, "v")
			v, err := semver.NewVersion(value)
			if err != nil {
				fmt.Printf("Skipping %s: unable to parse version\n", t.Name)
				continue
			}

			if _, ok := seen[v.String()]; ok {
				continue
			}
			seen[v.String()] = struct{}{}

			versions = append(versions, versionology.NewSimpleVersionFetcher(v))
		}
	}

	return versions, nil
}

func fetchTagsPage(owner, repo string, page int) ([]tag, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags?page=%d&per_page=100", owner, repo, page)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d for %s", resp.StatusCode, url)
	}

	var tags []tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	return tags, nil
}
