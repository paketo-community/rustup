# `gcr.io/paketo-community/rustup`

The Paketo Rustup Buildpack is a Cloud Native Buildpack that installs and executes `rustup` to install Rust.

## Behavior

If all of these conditions are met:

* Another buildpack requires `rust`
* `$BP_RUSTUP_ENABLED` is `true`

The buildpack will do the following:

* Contributes `rustup-init` to a layer marked `cache` with command on `$PATH`
* Executes `rustup-init` with the output written to a layer marked `build` and `cache` with installed commands on `$PATH`
* Executes `rustup` to install a Rust toolchain to a layer marked `build` and `cache` with installed commands on `$PATH`
* If `$BP_RUST_TARGET` is set, executes `rustup target add` to install an additional Rust target.

## Configuration
| Environment Variable      | Description                                                                                                                                                                                                                                                                  |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$BP_RUSTUP_ENABLED`      | Configure rustup to be enabled. This means that rustup will be used to install Rust. Default value is `true`. Set to false to use another Rust toolchain provider like [rust-dist](https://github.com/paketo-community/rust-dist).                                           |
| `$BP_RUST_TOOLCHAIN`      | Rust toolchain to install. Default `stable`. Other common values: `beta`, `nightly` or a specific versin number. Any [acceptable value for a toolchain](https://dev-doc.rust-lang.org/beta/edition-guide/rust-2018/rustup-for-managing-rust-versions.html) can be used here. |
| `$BP_RUST_PROFILE`        | Rust profile to install. Default `minimum`. Other acceptable values: `default`, `complete`. See [Rustup docs for profile](https://rust-lang.github.io/rustup/concepts/profiles.html).                                                                                        |
| `$BP_RUST_TARGET`         | Additional Rust target to install. Default ``, so nothing additional is installed. Run `rustup target list` to see what valid targets exist.                                                                                                                                 |
| `$BP_RUSTUP_INIT_VERSION` | Configure the version of rustup-init to install. It can be a specific version or a wildcard like `1.*`. It defaults to the latest `1.*` version.                                                                                                                             |
| `$BP_RUSTUP_INIT_LIBC`    | Configure the libc implementation used by the installed toolchain. Available options: `gnu` or `musl`. Defaults to `gnu` for compatiblity.                                                                                                                                   |

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
