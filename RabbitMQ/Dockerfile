# RabbitMQ 3.13 with management UI and pre-configured plugins
FROM rabbitmq:3.13-management

# Metadata
LABEL maintainer="mrengineer09"
LABEL description="RabbitMQ 3.13 with management UI, Prometheus monitoring, and pre-configured plugins"
LABEL version="3.13-plugins"

# Add plugin configuration into the image
# Plugins will be automatically enabled on startup
ADD enabled_plugins /etc/rabbitmq/enabled_plugins

# Add RabbitMQ configuration file
ADD rabbitmq.conf /etc/rabbitmq/rabbitmq.conf

# Add definitions file (exchanges, queues, bindings)
# This file includes user credentials and topology
ADD rabbitmq-definitions.json /etc/rabbitmq/definitions.json

# Health check - Uses HTTP endpoint from Management UI
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:15672/api/alarms || exit 1

# Expose standard ports
EXPOSE 5672 15672 25672

# Switch to rabbitmq user for security
USER rabbitmq

# Use default RabbitMQ entrypoint (no custom script needed)
# The official image will load definitions.json automatically
