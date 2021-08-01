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

			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "rustup"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "rustup-",
						"version": "1.24.3",
						"stacks":  []interface{}{"test-stack-id"},
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

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("rustup-"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("rustup-"))
		})
	})

	context("musl libc", func() {
		it.Before(func() {
			var err error

			ctx.Application.Path, err = ioutil.TempDir("", "build")
			Expect(err).NotTo(HaveOccurred())

			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "rustup"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "rustup-musl",
						"version": "1.24.3",
						"stacks":  []interface{}{"test-stack-id"},
					},
				},
			}
			ctx.StackID = "test-stack-id"

			Expect(os.Setenv("BP_RUSTUP_LIBC", "musl")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_RUSTUP_LIBC")).To(Succeed())
			Expect(os.RemoveAll(ctx.Application.Path)).To(Succeed())
		})

		it("contributes rustup", func() {
			result, err := build.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name()).To(Equal("rustup-musl"))

			Expect(result.BOM.Entries).To(HaveLen(1))
			Expect(result.BOM.Entries[0].Name).To(Equal("rustup-musl"))
		})
	})
}
