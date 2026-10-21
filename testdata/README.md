# Visual fixtures

Golden frames are deliberately not fabricated in the coding-only session.
The golden test fails if baselines are missing. To establish reviewed baselines:

1. Resolve dependencies with `go mod tidy`.
2. Render the gallery, review its layouts, then run:
   `go test ./internal/gallery -run TestGolden -update`.
3. Inspect and commit the resulting `golden/*.txt` files.
4. Run `make check` without the update flag.

Do not use the update flag to hide unintended differences.
