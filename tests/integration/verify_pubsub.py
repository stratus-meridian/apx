#!/usr/bin/env python3
"""
Simple script to verify Pub/Sub integration
Requires: pip install google-cloud-pubsub
"""

import os
import json
import time
from google.cloud import pubsub_v1

# Configuration
os.environ["PUBSUB_EMULATOR_HOST"] = "localhost:8085"
PROJECT_ID = "apx-dev"
TOPIC_NAME = "apx-requests-us-central1"
SUBSCRIPTION_NAME = "apx-workers-us-central1"

def main():
    print("=" * 50)
    print("Pub/Sub Verification Script")
    print("=" * 50)
    print()

    # Create subscriber client
    subscriber = pubsub_v1.SubscriberClient()
    subscription_path = subscriber.subscription_path(PROJECT_ID, SUBSCRIPTION_NAME)

    # Create subscription if it doesn't exist
    publisher = pubsub_v1.PublisherClient()
    topic_path = publisher.topic_path(PROJECT_ID, TOPIC_NAME)

    try:
        subscriber.create_subscription(
            request={"name": subscription_path, "topic": topic_path}
        )
        print(f"✅ Created subscription: {SUBSCRIPTION_NAME}")
    except Exception as e:
        print(f"ℹ️  Subscription exists or error: {e}")

    print(f"📡 Listening on subscription: {SUBSCRIPTION_NAME}")
    print()

    # Pull messages (blocking for 10 seconds)
    response = subscriber.pull(
        request={"subscription": subscription_path, "max_messages": 10},
        timeout=10.0
    )

    if not response.received_messages:
        print("❌ No messages received")
        return

    print(f"✅ Received {len(response.received_messages)} message(s)")
    print()

    for i, received_message in enumerate(response.received_messages):
        msg = received_message.message

        print(f"Message #{i+1}:")
        print(f"  Message ID: {msg.message_id}")
        print(f"  Publish Time: {msg.publish_time}")
        print(f"  Ordering Key: {msg.ordering_key}")
        print(f"  Attributes: {dict(msg.attributes)}")

        # Decode message data
        try:
            data = json.loads(msg.data.decode('utf-8'))
            print(f"  Data:")
            print(f"    Request ID: {data.get('request_id')}")
            print(f"    Tenant ID: {data.get('tenant_id')}")
            print(f"    Route: {data.get('route')}")
            print(f"    Method: {data.get('method')}")
            print(f"    Received At: {data.get('received_at')}")
        except Exception as e:
            print(f"  ⚠️  Failed to decode data: {e}")
            print(f"  Raw data: {msg.data}")

        print()

        # Acknowledge the message
        subscriber.acknowledge(
            request={"subscription": subscription_path, "ack_ids": [received_message.ack_id]}
        )

    print("=" * 50)
    print("✅ Verification complete!")
    print("=" * 50)

if __name__ == "__main__":
    main()
