# Use the latest version of Alpine Linux as the base image
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Set the working directory to /libredesk
WORKDIR /libredesk

# Copy necessary files
COPY libredesk .
COPY config.toml.sample config.toml

# Expose port 9000 for the application
EXPOSE 9000

# Set the default command to run the libredesk binary
CMD ["./libredesk"]