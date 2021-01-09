# Compile and package webhook
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./image/admission-webhook ./plugin/admission-webhook

# Docker build
docker build -t pkulkarni4/admission-webhook:v5 image/
