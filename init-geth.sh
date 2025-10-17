#!/bin/sh

set -e

# Check if the jwtsecret file exists
if [ -f "/tmp/jwtsecret" ]; then
  echo "Using jwtsecret from file"
elif [ ! -z "$JWT_SECRET" ]; then
  echo "Using jwtsecret from environment variable"
  echo "$JWT_SECRET" > /tmp/jwtsecret
else
  echo "JWT_SECRET environment variable is not set and jwtsecret file is not found"
  exit 1
fi

# Check if the genesis file is specified
if [ -z "$GENESIS_FILE" ]; then
  echo "GENESIS_FILE environment variable is not set"
  exit 1
fi

# Check if the genesis file exists
if [ ! -f "/$GENESIS_FILE" ]; then
  echo "Specified genesis file /$GENESIS_FILE does not exist"
  exit 1
fi

# Update the genesis file with the specified timestamp and mixHash if they are set
if [ ! -z "$GENESIS_TIMESTAMP" ]; then
  # Check if the timestamp is in hexadecimal format (starts with "0x")
  if [[ "$GENESIS_TIMESTAMP" =~ ^0x ]]; then
    echo "Using hexadecimal timestamp: $GENESIS_TIMESTAMP"
    timestamp_hex="$GENESIS_TIMESTAMP"
  else
    # Convert base 10 timestamp to hexadecimal
    echo "Converting base 10 timestamp to hexadecimal"
    timestamp_hex=$(printf "0x%x" "$GENESIS_TIMESTAMP")
  fi

  echo "Updating timestamp in genesis file"
  sed -i "s/\"timestamp\": \".*\"/\"timestamp\": \"$timestamp_hex\"/" "/$GENESIS_FILE"
else
  echo "GENESIS_TIMESTAMP environment variable is not set, using existing value in genesis file"
fi

if [ ! -z "$GENESIS_MIX_HASH" ]; then
  echo "Updating mixHash in genesis file"
  sed -i "s/\"mixHash\": \".*\"/\"mixHash\": \"$GENESIS_MIX_HASH\"/" "/$GENESIS_FILE"
else
  echo "GENESIS_MIX_HASH environment variable is not set, using existing value in genesis file"
fi

ENABLE_PREIMAGES=${ENABLE_PREIMAGES:-true}

# Build preimages flag for init if enabled
INIT_PREIMAGES_FLAG=""
if [ "$ENABLE_PREIMAGES" = "true" ]; then
  INIT_PREIMAGES_FLAG="--cache.preimages"
fi

# Check if the data directory is empty
if [ ! "$(ls -A /root/ethereum)" ]; then
  echo "Initializing new blockchain..."
  geth init $INIT_PREIMAGES_FLAG --state.scheme=hash --datadir /root/ethereum "/$GENESIS_FILE"
else
  echo "Blockchain already initialized."
fi

# Set default RPC gas cap if not provided
RPC_GAS_CAP=${RPC_GAS_CAP:-500000000}

# Set default cache size if not provided
CACHE_SIZE=${CACHE_SIZE:-25000}

# Set default auth RPC port if not provided
AUTH_RPC_PORT=${AUTH_RPC_PORT:-8551}

GC_MODE=${GC_MODE:-archive}

STATE_HISTORY=${STATE_HISTORY:-0}
TX_HISTORY=${TX_HISTORY:-0}

CACHE_GC=${CACHE_GC:-25}
CACHE_TRIE=${CACHE_TRIE:-15}


# Build override flags
OVERRIDE_FLAGS=""
if [ ! -z "$BLUEBIRD_TIMESTAMP" ]; then
  echo "Setting Bluebird fork timestamp to: $BLUEBIRD_TIMESTAMP"
  OVERRIDE_FLAGS="$OVERRIDE_FLAGS --override.bluebird=$BLUEBIRD_TIMESTAMP"
fi

# Build preimages flag if enabled
PREIMAGES_FLAG=""
if [ "$ENABLE_PREIMAGES" = "true" ]; then
  PREIMAGES_FLAG="--cache.preimages"
fi

# Log the configuration
echo "Starting geth with:"
echo "  GC Mode: $GC_MODE"
echo "  State History: $STATE_HISTORY blocks"
echo "  Transaction History: $TX_HISTORY blocks"
echo "  Cache Size: $CACHE_SIZE MB"
echo "  Preimages: $ENABLE_PREIMAGES"

# Start geth in background to fix IPC permissions
geth \
  --datadir /root/ethereum \
  --http \
  --http.addr "0.0.0.0" \
  --http.api "eth,net,web3,debug" \
  --http.vhosts="*" \
  --http.corsdomain="*" \
  --authrpc.addr "0.0.0.0" \
  --authrpc.vhosts="*" \
  --authrpc.port $AUTH_RPC_PORT \
  --authrpc.jwtsecret /tmp/jwtsecret \
  --nodiscover \
  --cache $CACHE_SIZE \
  --cache.gc $CACHE_GC \
  --cache.trie $CACHE_TRIE \
  $PREIMAGES_FLAG \
  --maxpeers 0 \
  --rpc.gascap $RPC_GAS_CAP \
  --syncmode full \
  --gcmode $GC_MODE \
  --rollup.disabletxpoolgossip \
  --rollup.enabletxpooladmission=false \
  --history.state $STATE_HISTORY \
  --history.transactions $TX_HISTORY $OVERRIDE_FLAGS &

# Capture the PID
GETH_PID=$!

# Wait for IPC socket to be created
echo "Waiting for IPC socket to be created..."
while [ ! -S /root/ethereum/geth.ipc ]; do
  sleep 0.5
done

# Make it world readable/writable
chmod 666 /root/ethereum/geth.ipc
echo "IPC socket permissions fixed to 666"
ls -la /root/ethereum/geth.ipc

# Bring geth back to foreground so Docker can track it properly
wait $GETH_PID
