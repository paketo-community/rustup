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
)

const (
	PlanEntryRustup = "rustup"
	PlanEntryRust   = "rust"
)

type Detect struct {
}

func (d Detect) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	cr, err := libpak.NewConfigurationResolver(context.Buildpack, nil)
	if err != nil {
		return libcnb.DetectResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	ok, err := d.rustupEnabled(cr)
	if err != nil {
		return libcnb.DetectResult{}, err
	}

	if !ok {
		return libcnb.DetectResult{Pass: false}, nil
	}

	return libcnb.DetectResult{
		Pass: true,
		Plans: []libcnb.BuildPlan{
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryRustup},
					{Name: PlanEntryRust},
				},
			},
			{
				Provides: []libcnb.BuildPlanProvide{
					{Name: PlanEntryRustup},
				},
			},
		},
	}, nil
}

func (d Detect) rustupEnabled(cr libpak.ConfigurationResolver) (bool, error) {
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
