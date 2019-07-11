SHELL := /bin/bash

all: keys vetpms-api metrics

keys:
	go run ./cmd/vetpms-admin/main.go keygen private.pem

admin:
	go run ./cmd/vetpms-admin/main.go --db-disable-tls=1 useradd admin@example.com gophers

migrate:
	go run ./cmd/vetpms-admin/main.go --db-disable-tls=1 migrate

seed: migrate
	go run ./cmd/vetpms-admin/main.go --db-disable-tls=1 seed

vetpms-api:
	docker build \
		-t gcr.io/vetpms-api/vetpms-api-amd64:1.0 \
		--build-arg PACKAGE_NAME=vetpms-api \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.
	docker system prune -f

metrics:
	docker build \
		-t gcr.io/vetpms-api/metrics-amd64:1.0 \
		--build-arg PACKAGE_NAME=metrics \
		--build-arg PACKAGE_PREFIX=sidecar/ \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.
	docker system prune -f

up:
	docker-compose up

down:
	docker-compose down

test:
	cd "$$GOPATH/src/github.com/os-foundry/vetpms"
	go test ./...

clean:
	docker system prune -f

stop-all:
	docker stop $(docker ps -aq)

remove-all:
	docker rm $(docker ps -aq)

#===============================================================================
# GKE

config:
	@echo Setting environment for vetpms-api
	gcloud config set project vetpms-api
	gcloud config set compute/zone us-central1-b
	gcloud auth configure-docker
	@echo ======================================================================

project:
	gcloud projects create vetpms-api
	gcloud beta billing projects link vetpms-api --billing-account=$(ACCOUNT_ID)
	gcloud services enable container.googleapis.com
	@echo ======================================================================

cluster:
	gcloud container clusters create vetpms-api-cluster --num-nodes=2 --machine-type=n1-standard-2
	gcloud compute instances list
	@echo ======================================================================

upload:
	docker push gcr.io/vetpms-api/vetpms-api-amd64:1.0
	docker push gcr.io/vetpms-api/metrics-amd64:1.0
	docker push gcr.io/vetpms-api/tracer-amd64:1.0
	@echo ======================================================================

database:
	kubectl create -f gke-deploy-database.yaml
	kubectl expose -f gke-expose-database.yaml --type=LoadBalancer
	@echo ======================================================================

services:
	kubectl create -f gke-deploy-vetpms-api.yaml
	kubectl expose -f gke-expose-vetpms-api.yaml --type=LoadBalancer
	@echo ======================================================================

shell:
	kubectl exec -it pod-name --container name -- /bin/bash
	@echo ======================================================================

status:
	gcloud container clusters list
	kubectl get nodes
	kubectl get pods
	kubectl get services vetpms-api
	@echo ======================================================================

delete:
	kubectl delete services vetpms-api
	kubectl delete deployment vetpms-api	
	gcloud container clusters delete vetpms-api-cluster
	gcloud projects delete vetpms-api
	docker image remove gcr.io/vetpms-api/vetpms-api-amd64:1.0
	docker image remove gcr.io/vetpms-api/metrics-amd64:1.0
	docker image remove gcr.io/vetpms-api/tracer-amd64:1.0
	@echo ======================================================================
