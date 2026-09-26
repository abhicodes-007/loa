#!/bin/bash
set -e

# We expect USER_ID and GROUP_ID to be passed as environment variables.
# Default to 1000 if not set.
USER_ID=${USER_ID:-1000}
GROUP_ID=${GROUP_ID:-1000}

# Create a group if the GID doesn't exist
if ! getent group "$GROUP_ID" > /dev/null 2>&1; then
    groupadd -g "$GROUP_ID" loa-group
else
    GROUP_NAME=$(getent group "$GROUP_ID" | cut -d: -f1)
fi

GROUP_NAME=${GROUP_NAME:-loa-group}

# Modify the 'loa' user to match the given UID and GID
usermod -u "$USER_ID" -g "$GROUP_ID" loa

# Ensure the loa user owns its home directory and the config mount
chown -R "loa:$GROUP_NAME" /home/loa || true

# If a command was passed, run it via gosu as the loa user
if [ "$#" -gt 0 ]; then
    exec gosu loa "$@"
else
    # Default to just starting the loa agent in the current directory
    exec gosu loa loa .
fi
