# Retying http forwarder

Act as a queue: will forward http-request to remote host and retry if remote host did not return success

Example:
curl --location --request POST "https://retryer-dot-retryer.appspot.com/post?HostToForwardTo=https://postman-echo.com"   --data "This is expected to be sent back as part of response body."

