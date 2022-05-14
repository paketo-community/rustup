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
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-community/rustup/rustup"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
)

func testRustupInit(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "rustup-layers")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes rustup-init", func() {
		dep := libpak.BuildpackDependency{
			URI:    "https://localhost/stub-rustup-init",
			SHA256: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		r := rustup.NewRustupInit(dep, dc)

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeFalse())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())
		Expect(filepath.Join(layer.Path, "bin", "stub-rustup-init")).To(BeARegularFile())

		stat, err := os.Stat(filepath.Join(layer.Path, "bin", "stub-rustup-init"))
		Expect(err).ToNot(HaveOccurred())
		Expect(stat.Mode().Perm().String()).To(Equal("-rwxr-xr-x"))
	})
}
