help:
	@echo "commands:"
	@echo "	make compile        Compiles the Elm code into JS (requires elm-make)"
	@echo "	make compile-watch  Compiles continuously on file changes (requires elm-live)"
	@echo "	make static         Bundle static assets into Go binary (requires esc)"
	@echo "	make test           Run frontend tests (requires elm-test)"
	@echo "	make test-watch     Runs frontend tests continuously on file changes (requires elm-test"

compile:
	elm-make src/Main.elm --output=scripts/console.js

compile-watch:
	elm-live src/Main.elm --output=scripts/console.js

static:
	esc -pkg main -ignore 'elm-*|.git*|.go$|src|tests' -o static.go .

test:
	elm-test

test-watch:
	elm-test --watch