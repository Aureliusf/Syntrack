.PHONY: build install test clean serve collect

build:
	go build -o syntrack .

install: build
	./syntrack collect

test:
	go test ./...

clean:
	rm -f syntrack usage.db

serve:
	./syntrack serve

collect:
	./syntrack collect

status:
	./syntrack status

history:
	./syntrack history

stats:
	./syntrack stats

cron-install:
	chmod +x scripts/install-cron.sh
	./scripts/install-cron.sh
