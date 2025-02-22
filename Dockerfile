# Use the latest version of Alpine Linux as the base image
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates

# Set the working directory to /libredesk
WORKDIR /libredesk

# Copy the libredesk binary to the working directory
COPY libredesk.bin .

# Expose port 9000 for the application
EXPOSE 9000

# Set the default command to run the libredesk binary
CMD ["./libredesk.bin"]