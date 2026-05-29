FROM alpine:3.18

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /libredesk

# Copy EVERYTHING into container
COPY . .

# Make binary executable
RUN chmod +x /libredesk/libredesk

# Expose app port
EXPOSE 9000

# Start Libredesk
CMD ["./libredesk"]