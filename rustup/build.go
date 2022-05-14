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
	"runtime"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, nil)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	if ok := cr.ResolveBool("BP_RUSTUP_ENABLED"); !ok {
		for _, entry := range context.Plan.Entries {
			result.Unmet = append(result.Unmet, libcnb.UnmetPlanEntry{Name: entry.Name})
		}
		return result, nil
	} else {
		// create a second time so the configuration is only printed after we know the buildpack should run
		cr, err = libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
		}

		dc, err := libpak.NewDependencyCache(context)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
		}
		dc.Logger = b.Logger

		dr, err := libpak.NewDependencyResolver(context)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
		}

		// install rustup-init
		v, _ := cr.Resolve("BP_RUSTUP_INIT_VERSION")
		libc, _ := cr.Resolve("BP_RUSTUP_INIT_LIBC")

		rustupInitDependency, err := dr.Resolve(fmt.Sprintf("rustup-init-%s", libc), v)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		rustupInit := NewRustupInit(rustupInitDependency, dc)
		rustupInit.Logger = b.Logger

		result.Layers = append(result.Layers, rustupInit)

		// make layer for cargo, which is installed by rust
		cargo := Cargo{}
		cargo.Logger = b.Logger
		result.Layers = append(result.Layers, cargo)

		// install rustup
		profile, _ := cr.Resolve("BP_RUST_PROFILE")
		rustup := NewRustup(rustupInitDependency.Version, profile)
		rustup.Logger = b.Logger

		result.Layers = append(result.Layers, rustup)

		// install rust
		rustVersion, _ := cr.Resolve("BP_RUST_TOOLCHAIN")
		additionalTarget := AdditionalTarget(cr, context.StackID)
		rust := NewRust(profile, rustVersion, additionalTarget)
		rust.Logger = b.Logger

		result.Layers = append(result.Layers, rust)
	}

	return result, nil
}

func AdditionalTarget(cr libpak.ConfigurationResolver, stack string) string {
	val, _ := cr.Resolve(("BP_RUST_TARGET"))
	if val != "" {
		return val
	}

	arch := "x86_64"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
	}

	libc := "gnu"
	if stack == libpak.TinyStackID {
		libc = "musl"
	}

	return fmt.Sprintf("%s-unknown-linux-%s", arch, libc)
}
