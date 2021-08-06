# `gcr.io/paketo-community/rustup`

The Paketo Rustup Buildpack is a Cloud Native Buildpack that installs and executes `rustup` to install Rust.

## Behavior

* Another buildpack requires `rustup`
* Another buildpack requires `rust`

The buildpack will do the following:

* Contributes Rustup to a layer marked `build` and `cache` with command on `$PATH`
* Executes `rustup` to install the latest version of Rust

## Configuration
| Environment Variable   | Description                                                                                                                                                                                                                                                                                                                                                              |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `$BP_RUSTUP_ENABLED`   | Configure rustup to be enabled. This means that rustup will be used to install Rust. Default value is `true`. Set to false to use another Rust toolchain provider like [rust-dist](https://github.com/paketo-community/rust-dist).                                                                                                                                       |
| `$BP_RUSTUP_VERSION`   | Configure the version of rustup to install. It can be a specific version or a wildcard like `1.*`. It defaults to the latest `1.*` version.                                                                                                                                                                                                                              |
| `$BP_RUSTUP_LIBC`      | Configure the libc implementation used by the installed toolchain. Available options: `gnu` or `musl`. Defaults to `gnu` for compatiblity.                                                                                                                                                                                                                               |
| `$BP_RUSTUP_INIT_ARGS` | Configure any additional arguments you'd like to pass to `rustup` when it's executed. Defaults to `-q -y --no-modify-path --default-toolchain=stable --profile=minimal`. The set of args `-q -y --no-modify-path` will always be used. Setting `--default-toolchain` or `--profile` will override the default values. Setting anything else will be appended to the end. |

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
