#!/bin/bash
set -e

echo "=== RabbitMQ Custom Entrypoint ==="
echo "RABBITMQ_DEFAULT_USER: ${RABBITMQ_DEFAULT_USER:-not set}"
echo "RABBITMQ_DEFAULT_PASS: ${RABBITMQ_DEFAULT_PASS:+***set***}"

# Start RabbitMQ in background
echo "Starting RabbitMQ in background..."
rabbitmq-server -detached &
RABBITMQ_PID=$!

# Wait for RabbitMQ to be ready
echo "Waiting for RabbitMQ to be ready..."
timeout=60
counter=0
until rabbitmqctl status >/dev/null 2>&1; do
    sleep 1
    counter=$((counter + 1))
    if [ $counter -ge $timeout ]; then
        echo "ERROR: RabbitMQ failed to start within $timeout seconds"
        exit 1
    fi
done
echo "RabbitMQ is ready!"

# Create user from environment variables if they exist
if [ -n "$RABBITMQ_DEFAULT_USER" ] && [ -n "$RABBITMQ_DEFAULT_PASS" ]; then
    echo "Creating user $RABBITMQ_DEFAULT_USER..."
    
    # Try to add user, ignore if already exists
    if rabbitmqctl add_user "$RABBITMQ_DEFAULT_USER" "$RABBITMQ_DEFAULT_PASS" 2>/dev/null; then
        echo "User $RABBITMQ_DEFAULT_USER created"
    else
        echo "User $RABBITMQ_DEFAULT_USER already exists, updating..."
        rabbitmqctl change_password "$RABBITMQ_DEFAULT_USER" "$RABBITMQ_DEFAULT_PASS"
    fi
    
    # Set permissions
    rabbitmqctl set_user_tags "$RABBITMQ_DEFAULT_USER" administrator
    rabbitmqctl set_permissions -p / "$RABBITMQ_DEFAULT_USER" ".*" ".*" ".*"
    echo "User $RABBITMQ_DEFAULT_USER configured successfully"
else
    echo "WARNING: RABBITMQ_DEFAULT_USER or RABBITMQ_DEFAULT_PASS not set"
    echo "Using guest user (NOT RECOMMENDED for production)"
fi

# Stop the detached server
echo "Stopping detached RabbitMQ..."
rabbitmqctl stop

# Wait a bit for clean shutdown
sleep 2

# Start RabbitMQ in foreground (as PID 1)
echo "Starting RabbitMQ in foreground..."
exec rabbitmq-server
