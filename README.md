# Retying http forwarder

Act as a queue: will forward http-request to remote host and retry if remote host did not return success

## deploy
  
    gcloud auth login
    gcloud config set project retryer
    gcloud app deploy app.yaml queue.yaml --quiet --version 3
    
## test

Example:

    curl --location --request POST "https://retryer-dot-retryer.appspot.com/post?HostToForwardTo=https://postman-echo.com"   --data "This is expected to be sent back as part of response body."

