# `gcr.io/paketo-community/rustup`

The Paketo Rustup Buildpack is a Cloud Native Buildpack that installs and executes `rustup` to install Rust.

## Behavior

* Another buildpack requires `rustup`
* Another buildpack requires `rust`

The buildpack will do the following:

* Contributes Rustup to a layer marked `build` and `cache` with command on `$PATH`
* Executes `rustup` to install the latest version of Rust

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
