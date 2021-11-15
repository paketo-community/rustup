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
	"strconv"

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

	if ok, err := b.rustupEnabled(cr); err != nil {
		return libcnb.BuildResult{}, err
	} else if ok {
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

		rustupInitDependency, err := dr.Resolve(fmt.Sprintf("rustup-%s", libc), v)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		rustupInit, be := NewRustupInit(rustupInitDependency, dc)
		rustupInit.Logger = b.Logger

		result.Layers = append(result.Layers, rustupInit)
		result.BOM.Entries = append(result.BOM.Entries, be)

		// make layer for cargo, which is installed by rust
		cargo := Cargo{}
		cargo.Logger = b.Logger
		result.Layers = append(result.Layers, cargo)

		// install rustup
		profile, _ := cr.Resolve("BP_RUST_PROFILE")
		rustup, be := NewRustup(rustupInitDependency.Version, profile)
		rustup.Logger = b.Logger

		result.Layers = append(result.Layers, rustup)
		// TODO: add when layer is emitting BOM
		// result.BOM.Entries = append(result.BOM.Entries, be)

		// install rust
		rustVersion, _ := cr.Resolve("BP_RUST_TOOLCHAIN")
		rust, be := NewRust(profile, rustVersion)
		rust.Logger = b.Logger

		result.Layers = append(result.Layers, rust)
		// TODO: add when layer is emitting BOM
		// result.BOM.Entries = append(result.BOM.Entries, be)
	}

	return result, nil
}

func (d Build) rustupEnabled(cr libpak.ConfigurationResolver) (bool, error) {
	val, _ := cr.Resolve("BP_RUSTUP_ENABLED")
	enable, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf(
			"invalid value '%s' for key '%s': expected one of [1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False]",
			val,
			"BP_RUSTUP_ENABLED",
		)
	}
	return enable, nil
}
