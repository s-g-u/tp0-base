#!/bin/bash

SERVER_CONTAINER="server"
SERVER_PORT=12345

TEST_MESSAGE="Test message"

RESPONSE=$(docker run --rm --network tp0_testing_net busybox:latest sh -c "echo '$TEST_MESSAGE' | nc $SERVER_CONTAINER $SERVER_PORT")

if [ "$RESPONSE" = "$TEST_MESSAGE" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
