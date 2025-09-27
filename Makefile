.PHONY: help
help: ## Yordam ko'rsatish
	@echo "Mavjud buyruqlar:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Ilovani ishga tushirish
	go run cmd/app/main.go

.PHONY: build
build: ## Binary yaratish
	CGO_ENABLED=0 GOOS=linux go build -o bin/sale-service cmd/app/main.go

# ==================== PROTO COMMANDS ====================

# Proto generation
.PHONY: proto
proto: ## Proto fayllarni generate qilish
	@echo "🚀 Proto generation boshlandi..."
	@chmod +x scripts/gen_proto.sh
	@bash scripts/gen_proto.sh $(PWD)

.PHONY: proto-clean
proto-clean: ## Generate qilingan proto fayllarni o'chirish
	@echo "🧹 Proto fayllarni tozalash..."
	@rm -rf genproto/
	@echo "✅ Tozalandi!"

.PHONY: proto-check
proto-check: ## Proto toollar borligini tekshirish
	@echo "🔍 Proto toollarni tekshirish..."
	@command -v protoc >/dev/null 2>&1 || { echo "❌ protoc topilmadi!"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "❌ protoc-gen-go topilmadi!"; exit 1; }
	@command -v protoc-gen-go-grpc >/dev/null 2>&1 || { echo "❌ protoc-gen-go-grpc topilmadi!"; exit 1; }
	@echo "✅ Barcha toollar o'rnatilgan!"
	@echo "   protoc: $$(protoc --version)"
	@echo "   Go: $$(go version)"
	
.PHONY: proto-all
proto-all: deps proto-check proto-clean proto ## Barchasini bajarish
	@echo "🎉 Proto generation to'liq bajarildi!"

# ==================== DATABASE MIGRATIONS ====================

.PHONY: migrate-up
migrate-up: ## Database migration ni ishga tushirish
	migrate -path migrations/postgres -database "postgresql://postgres:1234@localhost:5432/sale_db?sslmode=disable" up

.PHONY: migrate-down
migrate-down: ## Database migration ni bekor qilish
	migrate -path migrations/postgres -database "postgresql://postgres:1234@localhost:5432/sale_db?sslmode=disable" down

.PHONY: migrate-create
migrate-create: ## Yangi migration yaratish: make migrate-create name=add_users_table
	@if [ -z "$(name)" ]; then \
		echo "❌ Xatolik: Migration nomini kiriting!"; \
		echo "Misol: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations/postgres -seq $(name)

.PHONY: migrate-force
migrate-force: ## Migration version ni majburan o'rnatish: make migrate-force version=1
	@if [ -z "$(version)" ]; then \
		echo "❌ Xatolik: Version ni kiriting!"; \
		echo "Misol: make migrate-force version=1"; \
		exit 1; \
	fi
	migrate -path migrations/postgres -database "postgresql://postgres:1234@localhost:5432/sale_db?sslmode=disable" force $(version)

.PHONY: migrate-version
migrate-version: ## Joriy migration versiyasini ko'rsatish
	migrate -path migrations/postgres -database "postgresql://postgres:1234@localhost:5432/sale_db?sslmode=disable" version

# ==================== DOCKER COMMANDS ====================

.PHONY: docker-build
docker-build: ## Docker image yaratish
	docker build -t sale-service:latest -f build/Dockerfile .

.PHONY: docker-up
docker-up: ## Docker compose ni ishga tushirish
	docker compose -f build/docker-compose.yml up -d

.PHONY: docker-down
docker-down: ## Docker compose ni to'xtatish
	docker compose -f build/docker-compose.yml down

.PHONY: docker-logs
docker-logs: ## Docker logs ni ko'rish
	docker compose -f build/docker-compose.yml logs -f

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Docker ni qayta ishga tushirish

.PHONY: docker-clear
docker-clear: ## Docker ni tozalash - containerlarni to'xtatish va o'chirish
	docker stop $$(docker ps -aq)
	docker rm $$(docker ps -aq)

docker-clean: ## Docker containerlarni tozalash
	docker container prune -f

docker-clean-all: ## Barcha Docker resurslarini tozalash
	docker system prune -f

docker-clean-force: ## Kuchli tozalash - barcha containerlarni to'xtatish va o'chirish
	docker stop $$(docker ps -aq) 2>/dev/null || true
	docker rm $$(docker ps -aq) 2>/dev/null || true

docker-clean-volumes: ## Faqat volumelarni tozalash
	docker volume prune -f

docker-clean-networks: ## Faqat networklarni tozalash
	docker network prune -f

docker-clean-images: ## Faqat imagelarni tozalash
	docker image prune -af

# ==================== KAFKA COMMANDS ====================

.PHONY: kafka-create-topic
kafka-create-topic: ## Kafka topic yaratish
	docker exec -it kafka kafka-topics --create --topic product-events --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1

.PHONY: kafka-list-topics
kafka-list-topics: ## Kafka topiclarni ko'rish
	docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092

.PHONY: kafka-consume
kafka-consume: ## Kafka xabarlarni o'qish
	docker exec -it kafka kafka-console-consumer --topic product-events --from-beginning --bootstrap-server localhost:9092

.PHONY: kafka-produce
kafka-produce: ## Kafka ga xabar yuborish
	docker exec -it kafka kafka-console-producer --topic product-events --bootstrap-server localhost:9092

# ==================== TESTING ====================

.PHONY: test
test: ## Testlarni ishga tushirish
	go test -v -race ./...

.PHONY: test-cover
test-cover: ## Test coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: test-unit
test-unit: ## Unit testlar
	go test -v -race -short ./...

.PHONY: test-integration
test-integration: ## Integration testlar
	go test -v -race -run Integration ./...

# ==================== CODE QUALITY ====================

.PHONY: lint
lint: ## Kod sifatini tekshirish
	golangci-lint run

.PHONY: fmt
fmt: ## Kodni formatlash
	go fmt ./...

.PHONY: vet
vet: ## Go vet ishlatish
	go vet ./...

.PHONY: check
check: fmt vet lint ## Barcha tekshiruvlar

# ==================== DEPENDENCIES ====================

.PHONY: deps
deps: ## Dependencies ni o'rnatish
	@echo "📦 Dependencies o'rnatilmoqda..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go mod download
	@echo "✅ Dependencies o'rnatildi!"

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: deps-update
deps-update: ## Dependencies ni yangilash
	go get -u ./...
	go mod tidy

# ==================== CLEANUP ====================

.PHONY: clean
clean: ## Barcha build va cache fayllarni o'chirish
	@echo "🧹 Tozalash..."
	@rm -rf bin/
	@rm -rf genproto/
	@rm -rf coverage.out
	@rm -rf *.log
	@echo "✅ Tozalandi!"

# ==================== DATABASE UTILITIES ====================

.PHONY: db-create
db-create: ## Database yaratish
	-docker exec sale-postgres psql -U postgres -c "CREATE DATABASE sale_db;" 2>/dev/null || true
	@echo "✅ Database yaratish jarayoni tugatildi"

.PHONY: db-drop
db-drop: ## Database ni o'chirish
	docker exec -it sale-postgres psql -U postgres -c "DROP DATABASE IF EXISTS sale_db;"

.PHONY: db-reset
db-reset: db-drop db-create migrate-up ## Database ni reset qilish

.PHONY: db-connect
db-connect: ## PostgreSQL ga ulanish
	docker exec -it sale-postgres psql -U postgres -d sale_db

# ==================== FULL SETUP ====================

.PHONY: setup
setup: deps proto docker-up db-create migrate-up ## To'liq setup
	@echo "🎉 Setup to'liq bajarildi!"

.PHONY: dev
dev: setup run ## Development rejimda ishga tushirish

# ==================== QUICK COMMANDS ====================

.PHONY: up
up: docker-up migrate-up run ## Tez ishga tushirish

.PHONY: down
down: docker-down ## Hammasini to'xtatish

.PHONY: restart
restart: down up ## Qayta ishga tushirish