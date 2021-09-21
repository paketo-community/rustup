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
	"github.com/paketo-buildpacks/libpak/effect"
)

// Rustup will run `rustup-init` from the PATH and install `rustup`
//   It configures a default toolchain of `none`, so Rust isn't actually installed yet
type Rustup struct {
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Executor         effect.Executor
	Profile          string
}

func NewRustup(rustupInitVersion string, profile string) (Rustup, libcnb.BOMEntry) {
	return Rustup{
		LayerContributor: libpak.NewLayerContributor(
			"Rustup",
			map[string]interface{}{
				"rustupInitVersion": rustupInitVersion,
				"profile":           profile,
			},
			libcnb.LayerTypes{
				Build: true,
				Cache: true,
			}),
		Executor: effect.NewExecutor(),
		Profile:  profile,
	}, libcnb.BOMEntry{}
}

func (r Rustup) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	r.LayerContributor.Logger = r.Logger

	// configure env for the current buildpack
	if err := AppendToPath(filepath.Join(layer.Path, "bin")); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to set $PATH\n%w", err)
	}

	if err := os.Setenv("RUSTUP_HOME", layer.Path); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to set $RUSTUP_HOME\n%w", err)
	}

	layer, err := r.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		r.Logger.Body("Installing Rustup")

		if err := r.Executor.Execute(effect.Execution{
			Command: "rustup-init",
			Args: []string{
				"-q",
				"-y",
				"--no-modify-path",
				"--default-toolchain=none",
				fmt.Sprintf("--profile=%s", r.Profile),
			},
			Dir:    layer.Path,
			Stdout: bard.NewWriter(r.Logger.Logger.InfoWriter(), bard.WithIndent(3)),
			Stderr: bard.NewWriter(r.Logger.Logger.InfoWriter(), bard.WithIndent(3)),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to run rustup-init\n%w", err)
		}

		// remove `env` which collides with a buildpack spec defined folder
		if cargoHome, ok := os.LookupEnv("CARGO_HOME"); ok {
			if err := os.Remove(filepath.Join(cargoHome, "env")); err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to remove\n%w", err)
			}
		}

		layer.BuildEnvironment.Override("RUSTUP_HOME", layer.Path)

		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute Rust layer\n%w", err)
	}

	// TODO: populate & return BOM
	return layer, nil
}

func (r Rustup) Name() string {
	return r.LayerContributor.Name
}
