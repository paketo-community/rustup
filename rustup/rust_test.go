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

func testRust(t *testing.T, context spec.G, it spec.S) {
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
		Expect(os.MkdirAll(filepath.Join(cargoHome, "bin"), 0755))
		// we intentionally do not create a fake rustfmt here so we can test that it's OK if this file does not exist
		Expect(ioutil.WriteFile(filepath.Join(cargoHome, "bin", "cargo-fmt"), nil, 0644)).To(Succeed())

		Expect(os.Setenv("CARGO_HOME", cargoHome)).To(Succeed())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.Unsetenv("CARGO_HOME")).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes rust", func() {
		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			exec := args.Get(0).(effect.Execution)
			layer := filepath.Dir(filepath.Dir(exec.Command))
			Expect(ioutil.WriteFile(filepath.Join(layer, "env"), nil, 0644)).To(Succeed())
		})

		r, _ := rustup.NewRust("minimal", "1.2.3")
		r.Executor = executor

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execCheck := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execCheck.Command).To(Equal("rustup"))
		Expect(execCheck.Args).To(Equal([]string{"check"}))

		execShow := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execShow.Command).To(Equal("rustup"))
		Expect(execShow.Args).To(Equal([]string{"-q", "toolchain", "install", "--profile=minimal", "1.2.3"}))
		Expect(execShow.Dir).To(Equal(layer.Path))
	})

}
