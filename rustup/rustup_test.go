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
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
)

func testRustup(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx       libcnb.BuildContext
		executor  *mocks.Executor
		cargoHome string
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "rust-layers")
		Expect(err).NotTo(HaveOccurred())

		cargoHome, err = ioutil.TempDir("", "cargoHome")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(filepath.Join(cargoHome, "env"), nil, 0644)).To(Succeed())

		Expect(os.Setenv("CARGO_HOME", cargoHome)).To(Succeed())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.Unsetenv("CARGO_HOME")).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes rust", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustup"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustup 1.24.3 (2021-05-31)\ninfo: This is the version for the rustup toolchain manager, not the rustc compiler.\ninfo: The currently active `rustc` version is `rustc 1.60.0 (7737e0b5c 2022-04-04)`"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(ioutil.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		expectedArgs := []string{"-q", "-y", "--no-modify-path", "--default-toolchain=none", "--profile=minimal"}
		r := rustup.NewRustup("1.2.3", "minimal")
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execInit := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execInit.Command).To(Equal("rustup-init"))
		Expect(execInit.Args).To(Equal(expectedArgs))
		Expect(execInit.Dir).To(Equal(layer.Path))

		execVer := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execVer.Command).To(Equal("rustup"))
		Expect(execVer.Args).To(Equal([]string{"--version"}))

		Expect(filepath.Join(cargoHome, "env")).ToNot(BeAnExistingFile())
		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

}
