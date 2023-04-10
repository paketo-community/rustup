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

package rustup_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-community/rustup/rustup"
	"github.com/sclevine/spec"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		build rustup.Build
		ctx   libcnb.BuildContext
	)

	context("default libc", func() {
		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())

			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "rust"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "rustup-init-",
						"version": "1.24.3",
						"stacks":  []interface{}{"test-stack-id"},
					},
				},
				"configurations": []map[string]interface{}{
					{
						"name":        "BP_RUSTUP_ENABLED",
						"description": "use rustup to install Rust",
						"default":     "true",
						"build":       true,
					},
				},
			}
			ctx.StackID = "test-stack-id"
		})

		it.After(func() {
			Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		})

		it("contributes rustup", func() {
			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(4))
			Expect(result.Layers[0].Name()).To(Equal("rustup-init-"))
			Expect(result.Layers[1].Name()).To(Equal("Cargo"))
			Expect(result.Layers[2].Name()).To(Equal("Rustup"))
			Expect(result.Layers[3].Name()).To(Equal("Rust"))
		})

		context("$BP_RUSTUP_ENABLED is set", func() {
			context("to false", func() {
				it.Before(func() {
					Expect(os.Setenv("BP_RUSTUP_ENABLED", "false")).To(Succeed())
				})

				it.After(func() {
					Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
				})

				it("does not contribute", func() {
					result, err := build.Build(ctx)
					Expect(err).NotTo(HaveOccurred())

					Expect(result.Layers).To(HaveLen(0))
					Expect(result.Unmet).To(HaveLen(1))
					Expect(result.Unmet[0].Name).To(Equal("rust"))
				})
			})

			context("to true", func() {
				it.Before(func() {
					Expect(os.Setenv("BP_RUSTUP_ENABLED", "true")).To(Succeed())
				})

				it.After(func() {
					Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
				})

				it("contributes rustup", func() {
					result, err := build.Build(ctx)
					Expect(err).NotTo(HaveOccurred())

					Expect(result.Layers).To(HaveLen(4))
					Expect(result.Layers[0].Name()).To(Equal("rustup-init-"))
				})
			})

			context("to junk", func() {
				it.Before(func() {
					Expect(os.Setenv("BP_RUSTUP_ENABLED", "foobar")).To(Succeed())
				})

				it.After(func() {
					Expect(os.Unsetenv("BP_RUSTUP_ENABLED")).To(Succeed())
				})

				it("does not contribute", func() {
					result, err := build.Build(ctx)
					Expect(err).NotTo(HaveOccurred())

					Expect(result.Layers).To(HaveLen(0))
				})
			})
		})
	})

	context("musl libc", func() {
		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())

			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "rust"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "rustup-init-musl",
						"version": "1.24.3",
						"stacks":  []interface{}{"test-stack-id"},
					},
				},
				"configurations": []map[string]interface{}{
					{
						"name":        "BP_RUSTUP_ENABLED",
						"description": "use rustup to install Rust",
						"default":     "true",
						"build":       true,
					},
				},
			}
			ctx.StackID = "test-stack-id"

			Expect(os.Setenv("BP_RUSTUP_INIT_LIBC", "musl")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_RUSTUP_INIT_LIBC")).To(Succeed())
			Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		})

		it("contributes rustup", func() {
			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(4))
			Expect(result.Layers[0].Name()).To(Equal("rustup-init-musl"))
		})
	})

	context("pick additional target by stack", func() {
		it("picks gnu libc by default", func() {
			ctx.Buildpack.Metadata = map[string]interface{}{
				"configurations": []map[string]interface{}{},
			}
			ctx.StackID = "test-stack-id"

			cr, err := libpak.NewConfigurationResolver(ctx.Buildpack, nil)
			Expect(err).ToNot(HaveOccurred())

			target := rustup.AdditionalTarget(cr, libpak.BionicStackID)
			Expect(target).To(HaveSuffix("-unknown-linux-gnu"))
		})

		context("user value is set", func() {
			it("picks the user set value", func() {
				ctx.Buildpack.Metadata = map[string]interface{}{
					"configurations": []map[string]interface{}{
						{
							"name":        "BP_RUST_TARGET",
							"description": "additional rust target",
							"default":     "foo",
							"build":       true,
						},
					},
				}
				ctx.StackID = "test-stack-id"

				cr, err := libpak.NewConfigurationResolver(ctx.Buildpack, nil)
				Expect(err).ToNot(HaveOccurred())

				target := rustup.AdditionalTarget(cr, libpak.BionicStackID)
				Expect(target).To(Equal("foo"))
			})
		})

		it("picks musl for tiny stack", func() {
			ctx.Buildpack.Metadata = map[string]interface{}{
				"configurations": []map[string]interface{}{},
			}
			ctx.StackID = libpak.BionicTinyStackID

			cr, err := libpak.NewConfigurationResolver(ctx.Buildpack, nil)
			Expect(err).ToNot(HaveOccurred())

			target := rustup.AdditionalTarget(cr, libpak.BionicTinyStackID)
			Expect(target).To(HaveSuffix("-unknown-linux-musl"))
		})

		it("picks musl for static stack", func() {
			ctx.Buildpack.Metadata = map[string]interface{}{
				"configurations": []map[string]interface{}{},
			}
			ctx.StackID = libpak.JammyStaticStackID

			cr, err := libpak.NewConfigurationResolver(ctx.Buildpack, nil)
			Expect(err).ToNot(HaveOccurred())

			target := rustup.AdditionalTarget(cr, libpak.JammyStaticStackID)
			Expect(target).To(HaveSuffix("-unknown-linux-musl"))
		})
	})
}
