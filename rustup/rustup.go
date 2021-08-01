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
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/mattn/go-shellwords"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type Rustup struct {
	LayerContributor libpak.DependencyLayerContributor
	ConfigResolver   libpak.ConfigurationResolver
	Logger           bard.Logger
	Executor         effect.Executor
}

func NewRustup(dependency libpak.BuildpackDependency, cache libpak.DependencyCache, cr libpak.ConfigurationResolver) (Rustup, libcnb.BOMEntry) {
	contributor, entry := libpak.NewDependencyLayer(dependency, cache, libcnb.LayerTypes{
		Build: true,
		Cache: true,
	})
	return Rustup{
		LayerContributor: contributor,
		Executor:         effect.NewExecutor(),
	}, entry
}

func (r Rustup) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	r.LayerContributor.Logger = r.Logger

	return r.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		r.Logger.Header(color.BlueString("Rustup"))

		file := filepath.Join(layer.Path, "bin", filepath.Base(artifact.Name()))

		r.Logger.Bodyf("Copying to %s", filepath.Dir(file))

		if err := sherpa.CopyFile(artifact, file); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s\n%w", artifact.Name(), file, err)
		}

		if err := os.Chmod(file, 0755); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to chmod %s\n%w", file, err)
		}

		r.Logger.Body("Installing Rust")

		rustupInitArgs := []string{"-q", "-y", "--no-modify-path"}
		rustupInitVal, present := r.ConfigResolver.Resolve("BP_RUSTUP_INIT_ARGS")
		if present {
			additionalArgs, err := filterRustUpInitArgs(rustupInitVal)
			if err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to parse user specified args %s\n%w", rustupInitVal, err)
			}
			rustupInitArgs = append(rustupInitArgs, additionalArgs...)
		}
		rustupInitArgs = AddDefaults(rustupInitArgs)

		if err := r.Executor.Execute(effect.Execution{
			Command: filepath.Join(layer.Path, "bin", "rustup-init"),
			Args:    rustupInitArgs,
			Dir:     layer.Path,
			Env:     createEnviron(layer),
			// TODO: need an indent writer to indent each line
			Stdout: r.Logger.InfoWriter(),
			Stderr: r.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to run rustup-init\n%w", err)
		}
		r.Logger.Body("Ignore the message above from `rustup` to configure your shell. The buildpack will do this automatically.")

		// installer writes a file `env` to the layer, this conflicts with the spec
		//   which has an env directory under the layer for declaring env variables
		//   we are just renaming this env file so it doesn't conflict
		envFile := filepath.Join(layer.Path, "env")
		envSrcFile := fmt.Sprintf("%s-source", envFile)
		if err := os.Rename(envFile, envSrcFile); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to rename %s to %s\n%w", envFile, envSrcFile, err)
		}

		layer.BuildEnvironment.Override("RUSTUP_HOME", layer.Path)

		return layer, nil
	})
}

func (r Rustup) Name() string {
	return r.LayerContributor.LayerName()
}

func filterRustUpInitArgs(args string) ([]string, error) {
	argwords, err := shellwords.Parse(args)
	if err != nil {
		return nil, fmt.Errorf("unable to parse: %w", err)
	}

	var filteredArgs []string
	for _, arg := range argwords {
		if arg == "-q" || arg == "-y" || arg == "--no-modify-path" || arg == "--help" || arg == "-h" || arg == "-V" || arg == "--version" {
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}

	return filteredArgs, nil
}

func AddDefaults(args []string) []string {
	var profileSet = false
	var toolchainSet = false
	for _, arg := range args {
		if arg == "--profile" || strings.HasPrefix(arg, "--profile=") {
			profileSet = true
		}
		if arg == "--default-toolchain" || strings.HasPrefix(arg, "--default-toolchain=") {
			toolchainSet = true
		}
	}

	if !profileSet {
		args = append(args, "--profile=minimal")
	}

	if !toolchainSet {
		args = append(args, "--default-toolchain=stable")
	}

	return args
}

func createEnviron(layer libcnb.Layer) []string {
	env := os.Environ()

	// puts rustup & cargo binaries all in the layer's bin directory
	//   which means they'll all be on $PATH
	env = append(env, fmt.Sprintf("RUSTUP_HOME=%s", layer.Path))
	env = append(env, fmt.Sprintf("CARGO_HOME=%s", layer.Path))

	return env
}
