/*
 * Copyright 2018-2021 the original author or authors.
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

package rustup_test

import (
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-community/rustup/rustup"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx    libcnb.DetectContext
		detect rustup.Detect
	)

	it.Before(func() {
		ctx.Buildpack.Metadata = make(map[string]interface{})
		ctx.Buildpack.Metadata["configurations"] = []map[string]interface{}{
			{
				"name":        "BP_RUSTUP_ENABLED",
				"description": "use rustup to install Rust",
				"default":     "true",
				"build":       true,
			},
		}
	})

	it("includes default build plan options", func() {
		Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
			Pass: true,
			Plans: []libcnb.BuildPlan{
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "rustup"},
						{Name: "rust"},
					},
				},
				{
					Provides: []libcnb.BuildPlanProvide{
						{Name: "rustup"},
					},
				},
			},
		}))
	})

	context("$BP_RUSTUP_ENABLED is set", func() {
		context("to false", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_RUSTUP_ENABLED", "false")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
			})

			it("disables rustup plans", func() {
				Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
					Pass: false,
				}))
			})
		})

		context("to true", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_RUSTUP_ENABLED", "true")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
			})

			it("enables rustup plans", func() {
				Expect(detect.Detect(ctx)).To(Equal(libcnb.DetectResult{
					Pass: true,
					Plans: []libcnb.BuildPlan{
						{
							Provides: []libcnb.BuildPlanProvide{
								{Name: "rustup"},
								{Name: "rust"},
							},
						},
						{
							Provides: []libcnb.BuildPlanProvide{
								{Name: "rustup"},
							},
						},
					},
				}))
			})
		})

		context("to junk", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_RUSTUP_ENABLED", "foobar")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
			})

			it("fails", func() {
				_, err := detect.Detect(ctx)
				Expect(err).To(MatchError("invalid value 'foobar' for key 'BP_RUSTUP_ENABLED': expected one of [1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False]"))
			})
		})
	})
}
