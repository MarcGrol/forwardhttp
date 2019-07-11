# Retying http forwarder

This HTTP-service will as a persistent and retrying queue.<br/>
Upon receipt of a HTTP POST or PUT-request, the service will asynchronously forward the received HTTP request to a remote host.<br/>
When the remote host does not return a success, the request will be retried untill success or 
untill the retry scheme is exhausted.<br/>
The remote host is indicated by:
- the HTTP query parameeter "HostToForwardTo" or
- the HTTP-request-header "X-HostToForwardTo"

If required, asynchronous first delivery attmpts can be made. This functionality is triggered by:
- the HTTP query parameeter "TryFirst" or
- the HTTP-request-header "X-TryFirst"
   
## Deploy

Use gcloud command-line

    cd <project-root>
    gcloud auth login
    gcloud config set project <your-project-name>
    gcloud app deploy ./main/app.yaml  --quiet --version <your-version>
    
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

