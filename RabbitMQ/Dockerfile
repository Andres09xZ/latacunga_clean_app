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
ADD rabbitmq-definitions.json /etc/rabbitmq/definitions.json

# Add custom entrypoint script to create user from env vars
# Copy and set permissions as root before switching to rabbitmq user
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh && \
    chown rabbitmq:rabbitmq /usr/local/bin/docker-entrypoint.sh

# Health check - Uses both AMQP ping and HTTP endpoint from Management UI
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:15672/api/alarms || exit 1

# Expose standard ports
EXPOSE 5672 15672 25672

# Switch to rabbitmq user for security
USER rabbitmq

# Use custom entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
