#!/bin/bash

IMAGE_NAME="ghcr.io/ojparkinson/iracing-display:latest"

echo "Building Docker image..."
docker build -t $IMAGE_NAME .

echo "Pushing to GitHub Packages..."

echo $GITHUB_TOKEN | docker login ghcr.io -u ojparkinson --password-stdin
docker push $IMAGE_NAME

echo "Deployment complete! Watchtower will handle future updates automatically."

