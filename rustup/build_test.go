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
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-community/rustup/rustup"
	"github.com/sclevine/spec"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		build rustup.Build
		ctx   libcnb.BuildContext
	)

	context("writer indents", func() {
		it("indents by 2", func() {
			buf := bytes.Buffer{}
			writer := rustup.NewIndentWriter(2, &buf)

			_, err := writer.Write([]byte("hello\nworld!"))

			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal("  hello\n  world!"))
		})

		it("indents by 2", func() {
			buf := bytes.Buffer{}
			writer := rustup.NewIndentWriter(2, &buf)

			_, err := writer.Write([]byte(`foo
bar
baz
should
be
indented`))

			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(`  foo
  bar
  baz
  should
  be
  indented`))
		})

		it("indents by 2", func() {
			buf := bytes.Buffer{}
			writer := rustup.NewIndentWriter(2, &buf)

			err := effect.NewExecutor().Execute(effect.Execution{
				Command: "echo",
				Args:    []string{"-n", "a\nb\n\nc\n   d"},
				Stdout:  writer,
				Stderr:  writer,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal("  a\r\n  b\r\n  \r\n  c\r\n     d"))
		})

		it("nil writer", func() {
			writer := rustup.NewIndentWriter(2, nil)
			_, err := writer.Write([]byte("hello\nworld!"))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	context("default libc", func() {
		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())

			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "rust"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "rustup-",
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
			Expect(result.Layers[0].Name()).To(Equal("rustup-"))
			Expect(result.Layers[1].Name()).To(Equal("Cargo"))
			Expect(result.Layers[2].Name()).To(Equal("Rustup"))
			Expect(result.Layers[3].Name()).To(Equal("Rust"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("rustup-"))
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
					Expect(result.BOM.Entries).To(HaveLen(0))
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
					Expect(result.Layers[0].Name()).To(Equal("rustup-"))

					Expect(result.BOM.Entries).To(HaveLen(1))
					Expect(result.BOM.Entries[0].Name).To(Equal("rustup-"))
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
					_, err := build.Build(ctx)
					Expect(err).To(MatchError("invalid value 'foobar' for key 'BP_RUSTUP_ENABLED': expected one of [1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False]"))
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
						"id":      "rustup-musl",
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
			Expect(result.Layers[0].Name()).To(Equal("rustup-musl"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("rustup-musl"))
		})
	})
}
