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

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(ioutil.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		expectedArgs := []string{"-q", "-y", "--no-modify-path", "--default-toolchain=none", "--profile=minimal"}
		r, _ := rustup.NewRustup("1.2.3", "minimal")
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		executor := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(executor.Command).To(Equal("rustup-init"))
		Expect(executor.Args).To(Equal(expectedArgs))
		Expect(executor.Dir).To(Equal(layer.Path))

		Expect(filepath.Join(cargoHome, "env")).ToNot(BeAnExistingFile())
	})

}
