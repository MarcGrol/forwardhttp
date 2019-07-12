# Retying http forwarder

This HTTP-service will act as a persistent retrying queue.
Upon receipt of an HTTP POST-request or PUT-request, the service will asynchronously forward the received request to a remote host.
When the remote host does not return a success, the request will be retried untill success or 
untill the retry scheme is exhausted.
The remote host is indicated by:
- the HTTP query parameeter "HostToForwardTo" or
- the HTTP-request-header "X-HostToForwardTo"

If required, an synchronous first delivery attempt can be made. This functionality is triggered by:
- the HTTP query parameeter "TryFirst" or
- the HTTP-request-header "X-TryFirst"

## Install

    go get github.com/MarcGrol/forwardhttp
    
    cd ${GOPTH}/src/github.com/MarcGrol/forwardhttp
   
## Deploy

Use the gcloud command-line tool

    gcloud auth login # expect browser to pop-up for interactive login
    
    gcloud config set projectforwardhttp # or your own <project>
    
    gcloud app deploy ./main/app.yaml  --quiet --version 1
    
Service is now available at https://forwardhttp.appspot.com
    
## Test

https://forwardhttp.appspot.com


Example to test the interaction:

    curl -vvv \
        -X POST \
        --data "$(date): This is expected to be sent back as part of response body." \
        "https://<your-project-name>.appspot.com/post?HostToForwardTo=https://postman-echo.com"   

    curl -vvv \
        -X POST \
        --data "$(date): This is expected to be sent back as part of response body." \
        "https://forwardhttp.appspot.com/post?HostToForwardTo=https://postman-echo.com&TryFirst=true"  
        
## Create queue

Use gcloud command-line

    gcloud tasks queues create default \
        --max-attempts=3 \
        --max-concurrent-dispatches=3

    gcloud tasks queues update default
              --clear-max-attempts 
              --clear-max-retry-duration
              --clear-max-doublings 
              --clear-min-backoff
              --clear-max-backoff
              --clear-max-dispatches-per-second
              --clear-max-concurrent-dispatches
              --clear-routing-override         

