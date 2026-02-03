# Releasing

We use [GoReleaser](https://goreleaser.com/) to automate releases. The release workflow is triggered automatically when you push a version tag.

## Creating a Release

1. Check existing tags at [github.com/namespacelabs/spacectl/tags](https://github.com/namespacelabs/spacectl/tags)
2. Create a new tag with the next semantic version:
   ```bash
   git tag v0.1.0
   ```
3. Push the tag to GitHub:
   ```bash
   git push origin v0.1.0
   ```

The [release workflow](.github/workflows/release.yml) will automatically build and publish the release to GitHub.

## Testing Locally

To test GoReleaser configuration changes without publishing:

```bash
goreleaser release --clean --snapshot
```

This creates release artifacts in the `dist/` directory without uploading them.
