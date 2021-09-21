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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

// Rust will run `rustup` from the PATH to install a given toolchain
type Rust struct {
	LayerContributor libpak.LayerContributor
	Logger           bard.Logger
	Arguments        []string
	Executor         effect.Executor
	Toolchain        string
	Profile          string
}

func NewRust(profile string, toolchain string) (Rust, libcnb.BOMEntry) {
	return Rust{
		LayerContributor: libpak.NewLayerContributor(
			"Rust",
			map[string]interface{}{
				"toolchain": toolchain,
				"profile":   profile,
			},
			libcnb.LayerTypes{
				Build: true,
				Cache: true,
			}),
		Executor:  effect.NewExecutor(),
		Toolchain: toolchain,
		Profile:   profile,
	}, libcnb.BOMEntry{}
}

func (r Rust) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	r.LayerContributor.Logger = r.Logger

	// add `rustup check` to expected metadata if upstream rust changes, it won't match the layer metadata
	buf := bytes.Buffer{}
	if err := r.Executor.Execute(effect.Execution{
		Command: "rustup",
		Args:    []string{"check"},
		Stdout:  &buf,
		Stderr:  &buf,
	}); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to run `rustup check`: %s\n%w", buf.String(), err)
	}
	r.LayerContributor.ExpectedMetadata.(map[string]interface{})["installed"] = strings.TrimSpace(buf.String())

	layer, err := r.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		r.Logger.Body("Installing Rust")

		// remove these files because rustup forgets about them and thinks they are installed by someone else
		if cargoHome, ok := os.LookupEnv("CARGO_HOME"); ok {
			if err := os.Remove(filepath.Join(cargoHome, "bin", "rustfmt")); err != nil && !os.IsNotExist(err) {
				return libcnb.Layer{}, fmt.Errorf("unable to remove\n%w", err)
			}
			if err := os.Remove(filepath.Join(cargoHome, "bin", "cargo-fmt")); err != nil && !os.IsNotExist(err) {
				return libcnb.Layer{}, fmt.Errorf("unable to remove\n%w", err)
			}
		}

		if err := r.Executor.Execute(effect.Execution{
			Command: "rustup",
			Args: []string{
				"-q",
				"toolchain",
				"install",
				fmt.Sprintf("--profile=%s", r.Profile),
				r.Toolchain,
			},
			Dir:    layer.Path,
			Stdout: bard.NewWriter(r.Logger.Logger.InfoWriter(), bard.WithIndent(3)),
			Stderr: bard.NewWriter(r.Logger.Logger.InfoWriter(), bard.WithIndent(3)),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to run `rustup`\n%w", err)
		}

		return layer, nil
	})
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to contribute Rust layer\n%w", err)
	}

	// update metadata
	buf = bytes.Buffer{}
	if err := r.Executor.Execute(effect.Execution{
		Command: "rustup",
		Args:    []string{"check"},
		Stdout:  &buf,
		Stderr:  &buf,
	}); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to run `rustup check`: %s\n%w", buf.String(), err)
	}
	layer.Metadata["installed"] = strings.TrimSpace(buf.String())

	// TODO: populate & return BOM
	return layer, nil
}

func (r Rust) Name() string {
	return r.LayerContributor.Name
}
