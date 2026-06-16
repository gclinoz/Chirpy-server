.PHONY: all clean resq

all:
	go build -o out

resq:
	@curl -X POST http://localhost:8080/api/validate_chirp -H "Content-Type: application/json" -d '{"body": "I had something interesting for breakfast"}'
	@echo
	@curl -X POST http://localhost:8080/api/validate_chirp -H "Content-Type: application/json" -d '{"body": "I hear Mastodon is better than Chirpy. sharbert I need to migrate", "extra": "this should be ignored"}'
	@echo
	@curl -X POST http://localhost:8080/api/validate_chirp -H "Content-Type: application/json" -d '{"body": "I really need a kerfuffle to go to bed sooner, Fornax !"}'
	@echo
	@curl -X POST http://localhost:8080/api/validate_chirp -H "Content-Type: application/json" -d '{"body": "lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."}'
	@echo

clean:
	rm ./out
