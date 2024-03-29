[![BCH compliance](https://bettercodehub.com/edge/badge/MarcGrol/forwardhttp?branch=master)](https://bettercodehub.com/)


# Retying http forwarder

This HTTP-service will act as a persistent retrying queue.
Upon receipt of an HTTP POST, PUT and DELETE-requests, the service will asynchronously forward the received request to a remote host.
When the remote host does not return a success, the request will be retried untill success or 
untill the retry scheme is exhausted.
The remote host is indicated by:
- the HTTP query parameter "HostToForwardTo" or
- the HTTP-request-header "X-HostToForwardTo"

If required, an synchronous first delivery attempt can be made. This functionality is triggered by:
- the HTTP query parameter "TryFirst" or
- the HTTP-request-header "X-TryFirst"

## Install

    go get github.com/MarcGrol/forwardhttp
    
    cd ${GOPTH}/src/github.com/MarcGrol/forwardhttp
   
## Deploy

Use the gcloud command-line tool

    gcloud auth login # expect browser to pop-up for interactive login
    
    gcloud config set project forwardhttp # or your own <gcloud-project>
    
    gcloud app deploy ./main/app.yaml  --quiet --version 1
    
Service is now available at https://forwardhttp.appspot.com (or your own https://<gcloud-project>.appspot.com).
    
        
## Create queue

Use gcloud command-line

    gcloud tasks queues create default \
        --max-attempts=10 \
        --max-concurrent-dispatches=5 # prevent overloading remote system


## Test

https://forwardhttp.appspot.com


Example to test the interaction:

    curl -vvv \
        --request POST \
        --data "This is expected to be sent back as part of response body." \
        "https://forwardhttp.appspot.com/post?HostToForwardTo=postman-echo.com&TryFirst=false"   
