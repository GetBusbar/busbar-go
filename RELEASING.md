# Releasing busbar-go

Development happens on the private GitLab repository. GitHub is a public source mirror;
GitHub Actions, scheduled regeneration, and tag-triggered publication workflows are retired.

To update the client, take `openapi.json` from the intended qualified Busbar source revision,
run `make generate`, and validate the generated client with the repository's checks. Commit the
specification and generated client together on `dev`, recording the source revision.

Release promotion and tagging require an explicit release request through the internal pipeline.
A local commit or a push to the public mirror does not perform release validation or publication.
