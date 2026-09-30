# Lockscreen partner allocation case study

Start with the [command center](docs/COMMAND_CENTER.md) for the problem, decisions, and work history. The submission contains a [ranked RFC review](docs/REVIEW.md), a [production design](docs/DESIGN.md), a [Go proof of concept](poc/README.md), and [AI use notes](docs/AI_NOTES.md).

The POC tests selection across three mock partners, deadlines, failures, and fallback. From `poc/`, run:

```sh
go test ./...
go run ./cmd/poc -scenario=normal -n=3 -budget=80ms -runs=30
```

Its value figures use assumed click probabilities and are not measured revenue. The original exercise PDFs were supplied separately and are not included here.
