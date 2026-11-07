# Releasing busbar-go

Release automation runs in the internal pipeline. GitHub Actions, scheduled regeneration,
and tag-triggered publication workflows are retired.

To update the client, take `openapi.json` from the intended qualified Busbar source revision,
run `make generate`, and validate the generated client with the repository's checks. Commit the
specification and generated client together on `dev`, recording the source revision.

Release promotion and tagging require an explicit release request through the internal pipeline.
A local commit or a GitHub push does not perform release validation or publication.
