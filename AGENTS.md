# Project conventions

For application fixes and features, bump the patch version and build number as
part of the change. Keep `internal/version/version.go`,
`cmd/ansichat-gui/FyneApp.toml`, and
`cmd/ansichat-gui/AndroidManifest.xml` in sync. Update the GUI documentation
version and add release notes in `README.md`. Run `go test ./internal/version`
to verify the metadata. If the current change already includes a version bump,
reuse it rather than bumping again for follow-up edits to the same change.
