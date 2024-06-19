# quick-sqs-scanner

## What is this?

This is a quick program I wrote to test out SNS/SQS interactions via JSON.  You give it the _name_ of an SQS queue to read from,
and it will sit receiving messages from that queue.

For each message received, it will try to:
- decode the message body as JSON
- look for a `Message` string field within the JSON (this is the format of the SNS wrapper)
- if such a field is found, it will then print that to the console as coloured JSON (via https://github.com/TylerBrock/colorjson)

## How do I build/run it?

- You need Go installed (to build it), at least version 1.20 (1.21 or higher recommended)
- `go build`
- `./quick-sqs-scanner -help`

Alternatively, pick up the zipfile from Releases - nothing extra needed to install, just pick the right executable for your platform and off you go!
