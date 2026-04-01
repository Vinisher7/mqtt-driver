-include .env
export

migrate-create:
	migrate create -ext pgsql -dir $(MAKEFILE_DIR) -seq $(n)

migrate-up: 
	migrate -path $(MAKEFILE_DIR) -database $(MAKEFILE_DB_STRING) up $(n)

migrate-down: 
	migrate -path $(MAKEFILE_DIR) -database $(MAKEFILE_DB_STRING) down $(n)

migrate-version:
	migrate -path $(MAKEFILE_DIR) -database $(MAKEFILE_DB_STRING) version

migrate-goto:
	migrate -path $(MAKEFILE_DIR) -database $(MAKEFILE_DB_STRING) goto $(v)

migrate-force:
	migrate -path $(MAKEFILE_DIR) -database $(MAKEFILE_DB_STRING) force $(v)
