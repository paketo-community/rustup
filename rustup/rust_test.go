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

// fakeExitError is an error that reports a process exit code, mirroring the
// behavior of *exec.ExitError returned when a command exits non-zero.
type fakeExitError struct {
	code int
}

func (e fakeExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.code)
}

func (e fakeExitError) ExitCode() int {
	return e.code
}

func testRust(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx       libcnb.BuildContext
		executor  *mocks.Executor
		cargoHome string
		appPath   string
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = os.MkdirTemp("", "rust-layers")
		Expect(err).NotTo(HaveOccurred())

		appPath, err = os.MkdirTemp("", "app-dir")
		Expect(err).NotTo(HaveOccurred())

		cargoHome, err = os.MkdirTemp("", "cargoHome")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(cargoHome, "bin"), 0755))
		// we intentionally do not create a fake rustfmt here so we can test that it's OK if this file does not exist
		Expect(os.WriteFile(filepath.Join(cargoHome, "bin", "cargo-fmt"), nil, 0644)).To(Succeed())

		Expect(os.Setenv("CARGO_HOME", cargoHome)).To(Succeed())

		executor = &mocks.Executor{}
	})

	it.After(func() {
		Expect(os.Unsetenv("CARGO_HOME")).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
		Expect(os.RemoveAll(appPath)).To(Succeed())
	})

	it("contributes rust", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustc"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustc 1.2.3 (53cb7b09b 2021-06-17)\n"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		r := rustup.NewRust("minimal", "1.2.3", "", "", false, false)
		r.Executor = executor

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

		execVer := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(execVer.Command).To(Equal("rustc"))
		Expect(execVer.Args).To(Equal([]string{"--version"}))

		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

	it("contributes rust and a target", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustc"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustc 1.2.3 (53cb7b09b 2021-06-17)\n"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		r := rustup.NewRust("minimal", "1.2.3", "foo", "", false, false)
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execCheck := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execCheck.Command).To(Equal("rustup"))
		Expect(execCheck.Args).To(Equal([]string{"check"}))

		execToolchain := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execToolchain.Command).To(Equal("rustup"))
		Expect(execToolchain.Args).To(Equal([]string{"-q", "toolchain", "install", "--profile=minimal", "1.2.3"}))
		Expect(execToolchain.Dir).To(Equal(layer.Path))

		execTarget := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(execTarget.Command).To(Equal("rustup"))
		Expect(execTarget.Args).To(Equal([]string{"-q", "target", "add", "--toolchain=1.2.3", "foo"}))
		Expect(execTarget.Dir).To(Equal(layer.Path))

		execVer := executor.Calls[3].Arguments[0].(effect.Execution)
		Expect(execVer.Command).To(Equal("rustc"))
		Expect(execVer.Args).To(Equal([]string{"--version"}))

		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

	it("contributes rust and a target from rust-toolchain.toml", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		toolchainFilePath := filepath.Join(appPath, "rust-toolchain.toml")
		Expect(os.WriteFile(toolchainFilePath, []byte("foo"), 0644)).To(Succeed())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustc"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustc 1.2.3 (53cb7b09b 2021-06-17)\n"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		r := rustup.NewRust("minimal", "1.2.3", "foo", toolchainFilePath, false, false)
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execCheck := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execCheck.Command).To(Equal("rustup"))
		Expect(execCheck.Args).To(Equal([]string{"check"}))

		execDefault := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execDefault.Command).To(Equal("rustup"))
		Expect(execDefault.Args).To(Equal([]string{"-q", "default", "1.2.3"}))
		Expect(execDefault.Dir).To(Equal(layer.Path))

		execShow := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(execShow.Command).To(Equal("rustup"))
		Expect(execShow.Args).To(Equal([]string{"-q", "show"}))
		Expect(execShow.Dir).To(Equal(layer.Path))

		execTarget := executor.Calls[3].Arguments[0].(effect.Execution)
		Expect(execTarget.Command).To(Equal("rustup"))
		Expect(execTarget.Args).To(Equal([]string{"-q", "target", "add", "--toolchain=1.2.3", "foo"}))
		Expect(execTarget.Dir).To(Equal(layer.Path))

		execVer := executor.Calls[4].Arguments[0].(effect.Execution)
		Expect(execVer.Command).To(Equal("rustc"))
		Expect(execVer.Args).To(Equal([]string{"--version"}))

		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

	it("contributes rust and a target from rust-toolchain.toml and from env variable", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		toolchainFilePath := filepath.Join(appPath, "rust-toolchain.toml")
		Expect(os.WriteFile(toolchainFilePath, []byte("foo"), 0644)).To(Succeed())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustc"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustc 1.2.3 (53cb7b09b 2021-06-17)\n"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		r := rustup.NewRust("minimal", "1.2.3", "foo", toolchainFilePath, true, true)
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execCheck := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execCheck.Command).To(Equal("rustup"))
		Expect(execCheck.Args).To(Equal([]string{"check"}))

		execDefault := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execDefault.Command).To(Equal("rustup"))
		Expect(execDefault.Args).To(Equal([]string{"-q", "default", "1.2.3"}))
		Expect(execDefault.Dir).To(Equal(layer.Path))

		execShow := executor.Calls[2].Arguments[0].(effect.Execution)
		Expect(execShow.Command).To(Equal("rustup"))
		Expect(execShow.Args).To(Equal([]string{"-q", "show"}))
		Expect(execShow.Dir).To(Equal(layer.Path))

		execToolchain := executor.Calls[3].Arguments[0].(effect.Execution)
		Expect(execToolchain.Command).To(Equal("rustup"))
		Expect(execToolchain.Args).To(Equal([]string{"-q", "toolchain", "install", "--profile=minimal", "1.2.3"}))
		Expect(execToolchain.Dir).To(Equal(layer.Path))

		execTarget := executor.Calls[4].Arguments[0].(effect.Execution)
		Expect(execTarget.Command).To(Equal("rustup"))
		Expect(execTarget.Args).To(Equal([]string{"-q", "target", "add", "--toolchain=1.2.3", "foo"}))
		Expect(execTarget.Dir).To(Equal(layer.Path))

		execVer := executor.Calls[5].Arguments[0].(effect.Execution)
		Expect(execVer.Command).To(Equal("rustc"))
		Expect(execVer.Args).To(Equal([]string{"--version"}))

		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

	it("contributes rust when an update is available", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		// Starting with rustup 1.29.0, `rustup check` exits with code 100 when an
		// update is available. That must not fail the build, it should just
		// invalidate the layer cache and cause rust to be reinstalled.
		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Command == "rustup" && len(ex.Args) == 1 && ex.Args[0] == "check"
		})).Return(fakeExitError{100}).Run(func(args mock.Arguments) {
			ex := args.Get(0).(effect.Execution)
			_, err := ex.Stdout.Write([]byte("stable-x86_64-unknown-linux-gnu - update available: 1.98.1 -> 1.99.0\n"))
			Expect(err).ToNot(HaveOccurred())
		})

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Args[0] == "--version" && ex.Command == "rustc"
		})).Return(func(ex effect.Execution) error {
			_, err := ex.Stdout.Write([]byte("rustc 1.99.0 (b940084d7 2026-09-28)\n"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})

		executor.On("Execute", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(layer.Path, "env"), nil, 0644)).To(Succeed())
		})

		r := rustup.NewRust("minimal", "stable", "", "", false, false)
		r.Executor = executor

		layer, err = r.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.LayerTypes.Build).To(BeTrue())
		Expect(layer.LayerTypes.Cache).To(BeTrue())
		Expect(layer.LayerTypes.Launch).To(BeFalse())

		execCheck := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(execCheck.Command).To(Equal("rustup"))
		Expect(execCheck.Args).To(Equal([]string{"check"}))

		execInstall := executor.Calls[1].Arguments[0].(effect.Execution)
		Expect(execInstall.Command).To(Equal("rustup"))
		Expect(execInstall.Args).To(Equal([]string{"-q", "toolchain", "install", "--profile=minimal", "stable"}))
		Expect(execInstall.Dir).To(Equal(layer.Path))

		Expect(layer.SBOMPath(libcnb.SyftJSON)).To(BeARegularFile())
	})

	it("fails when `rustup check` fails", func() {
		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		executor.On("Execute", mock.MatchedBy(func(ex effect.Execution) bool {
			return ex.Command == "rustup" && len(ex.Args) == 1 && ex.Args[0] == "check"
		})).Return(fakeExitError{1})

		r := rustup.NewRust("minimal", "stable", "", "", false, false)
		r.Executor = executor

		_, err = r.Contribute(layer)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("unable to run `rustup check`")))
	})
}
