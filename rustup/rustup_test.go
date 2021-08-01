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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-community/rustup/rustup"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
)

func testRustup(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx      libcnb.BuildContext
		executor *mocks.Executor
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "rustup-layers")
		Expect(err).NotTo(HaveOccurred())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes rustup", func() {
		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			exec := args.Get(0).(effect.Execution)
			layer := filepath.Dir(filepath.Dir(exec.Command))
			Expect(ioutil.WriteFile(filepath.Join(layer, "env"), nil, 0644)).To(Succeed())
		})

		dep := libpak.BuildpackDependency{
			URI:    "https://localhost/stub-rustup-init",
			SHA256: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		logger := bard.NewLogger(ioutil.Discard)
		cr, err := libpak.NewConfigurationResolver(ctx.Buildpack, &logger)
		Expect(err).ToNot(HaveOccurred())

		r, _ := rustup.NewRustup(dep, dc, cr)
		r.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())
		Expect(filepath.Join(layer.Path, "bin", "stub-rustup-init")).To(BeARegularFile())

		stat, err := os.Stat(filepath.Join(layer.Path, "bin", "stub-rustup-init"))
		Expect(err).ToNot(HaveOccurred())
		Expect(stat.Mode().Perm().String()).To(Equal("-rwxr-xr-x"))

		executor := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(executor.Command).To(Equal(filepath.Join(layer.Path, "bin", "rustup-init")))
		Expect(executor.Args).To(Equal([]string{"-q", "-y", "--no-modify-path", "--profile=minimal", "--default-toolchain=stable"}))
		Expect(executor.Dir).To(Equal(layer.Path))
	})

	context("proper defaults", func() {
		it("sets defaults", func() {
			res := rustup.AddDefaults([]string{"--something"})

			Expect(res).To(HaveLen(3))
			Expect(res[0]).To(Equal("--something"))
			Expect(res[1]).To(Equal("--profile=minimal"))
			Expect(res[2]).To(Equal("--default-toolchain=stable"))
		})

		it("sets default profile with equals sign", func() {
			res := rustup.AddDefaults([]string{"--default-toolchain=nightly"})

			fmt.Println("args:", res)
			Expect(res).To(HaveLen(2))
			Expect(res[0]).To(Equal("--default-toolchain=nightly"))
			Expect(res[1]).To(Equal("--profile=minimal"))
		})

		it("sets default profile", func() {
			res := rustup.AddDefaults([]string{"--default-toolchain", "nightly"})

			Expect(res).To(HaveLen(3))
			Expect(res[0]).To(Equal("--default-toolchain"))
			Expect(res[1]).To(Equal("nightly"))
			Expect(res[2]).To(Equal("--profile=minimal"))
		})

		it("sets default toolchain with equals sign", func() {
			res := rustup.AddDefaults([]string{"--profile=default"})

			Expect(res).To(HaveLen(2))
			Expect(res[0]).To(Equal("--profile=default"))
			Expect(res[1]).To(Equal("--default-toolchain=stable"))
		})

		it("sets default toolchain", func() {
			res := rustup.AddDefaults([]string{"--profile", "default"})

			Expect(res).To(HaveLen(3))
			Expect(res[0]).To(Equal("--profile"))
			Expect(res[1]).To(Equal("default"))
			Expect(res[2]).To(Equal("--default-toolchain=stable"))
		})
	})

}
