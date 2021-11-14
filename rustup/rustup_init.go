/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rustup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

// RustupInit will handle installing the `rustup-init` tool & adding it to the PATH
type RustupInit struct {
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
}

func NewRustupInit(dependency libpak.BuildpackDependency, cache libpak.DependencyCache) (RustupInit, libcnb.BOMEntry) {
	contributor, entry := libpak.NewDependencyLayer(dependency, cache, libcnb.LayerTypes{
		Cache: true,
	})
	return RustupInit{
		LayerContributor: contributor,
	}, entry
}

func (r RustupInit) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	r.LayerContributor.Logger = r.Logger

	if err := os.Setenv("PATH", sherpa.AppendToEnvVar("PATH", ":", filepath.Join(layer.Path, "bin"))); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to set $PATH\n%w", err)
	}

	return r.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		file := filepath.Join(layer.Path, "bin", filepath.Base(artifact.Name()))

		r.Logger.Bodyf("Copying to %s", filepath.Dir(file))

		if err := sherpa.CopyFile(artifact, file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s\n%w", artifact.Name(), file, err)
		}

		if err := os.Chmod(file, 0755); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to chmod %s\n%w", file, err)
		}

		return layer, nil
	})
}

func (r RustupInit) Name() string {
	return r.LayerContributor.LayerName()
}
