#!/bin/bash

# Set variables
IMAGE_NAME="ghcr.io/ojparkinson/iracing-telemetryservice:latest"

# Build the image locally (on your more powerful machine)
echo "Building Docker image..."
docker build -t $IMAGE_NAME .

# Log in and push to GitHub Packages
echo "Pushing to GitHub Packages..."
# Use a GitHub Personal Access Token with appropriate permissions
echo $GITHUB_TOKEN | docker login ghcr.io -u ojparkinson --password-stdin
docker push $IMAGE_NAME


echo "Deployment complete! Watchtower will handle future updates automatically."

