# Stock Ticker Service

A simple Go application that fetches daily stock data from Alpha Vantage and caches it in a Redis database. This service exposes an HTTP endpoint to retrieve stock prices along with their average over a specified number of days.

## Features

- Fetches daily stock data for a given symbol from Alpha Vantage.
- Caches the data in Redis to improve performance and reduce impact of rate limiting.
- Provides an HTTP API to retrieve stock closing prices and their average.

## Prerequisites

Before you begin, ensure you have the following installed:

- [Docker](https://www.docker.com/get-started) (for containerization)
- [Docker Compose](https://docs.docker.com/compose/) (for managing multi-container applications)

## Environment Variables

The application uses several environment variables that must be set in the `docker-compose.yaml` file:

- `SYMBOL`: The stock symbol to fetch data for (e.g., `AAPL`).
- `NDAYS`: The number of days to retrieve stock prices for.
- `API_KEY`: Your API key for accessing Alpha Vantage.
- `REDIS_ADDR`: The address of the Redis service (default is `redis:6379`).

## Getting Started

1. **Clone to Repository**

   ```bash
   git clone https://github.com/ckav370/stock-ticker.git
   cd stock-ticker
   ```

2. **Create a .env File**

You can create a .env file in the root directory to specify your environment variables instead of editing the docker-compose.yaml file directly while also reducing the risk of commiting secrets to source code. For example:

```
SYMBOL=AAPL
NDAYS=5
API_KEY=$API_KEY
REDIS_ADDR=redis:6379
```

3. **Build and Run the Application**

Use Docker Compose to build the latest version of the image and run the application and Redis service:

```
docker-compose up --build
```

4. **Access the service**

Once the service is running, it will accessible at `localhost`. Exanple:

```
➜  curl http://localhost:8080/stock\?symbol\=AAPL\&ndays\=10

{"average_price":224.88000000000002,"closing_prices":[{"date":"2024-06-26","close":213.25},{"date":"2024-07-09","close":228.68},{"date":"2024-07-16","close":234.82},{"date":"2024-09-04","close":220.85},{"date":"2024-10-04","close":226.8}]}
```

The service will return a JSON response containing the closing prices and their average. For example:

```
{"average_price":224.88000000000002,"closing_prices":[{"date":"2024-06-26","close":213.25},{"date":"2024-07-09","close":228.68},{"date":"2024-07-16","close":234.82},{"date":"2024-09-04","close":220.85},{"date":"2024-10-04","close":226.8}]}
```

5. **Stopping the service**

To stop the services, run:

```
docker-compose down
```

This command will stop and remove all the containers defined in `docker-compose.yaml` file.


## Continuous Deployment

There is an example Github Workflow directory with some basic CI workflows which would be the minimum expected when productionising a real-world applications. 


## K8s Deployment

To deploy on Kubernetes, there are several dependencies required:

- [External Secrets Operator](https://external-secrets.io/latest/) (assuming a Cloud provider backend)
- [Ingress Nginx](https://github.com/kubernetes/ingress-nginx) (open source ingress controller, assuming the load balancer type in NLB and path based routing). The design also assumes TLS termination at the load balancer controller.

Ensure the `secret.yaml	` is using the External Secret secret version as this is a prefered method of secret management for a hosted K8s cluster due to ease of integration with Cloud providers, improved security and ease of use.

To apply the manifests:

```
 k apply -f manifests
```

## Kind Deployment


Below are instructions to deploy to a Kind cluster:

1. **Install Ingress Nginx**

```
helm install my-release oci://ghcr.io/nginxinc/charts/nginx-ingress --version 1.4.0
```

2. **Secrets**

Insure the `secret.yaml` is using the default `Secret` only

3. **Create Cluster**

```
kind delete cluster
```
4. **Apply Manifestsr**

```
 k apply -f manifests

```

## Productionisation

There are severval important factors that need to be considered before releasing an application such as this into a production environment. These include:

* The minimum replica count should be 3 to ensure 1 replica per AZ (best practise assumes 3 subnets, each in a different AZ)
* Resource requests and limits need to be baselined and stress tested against some degree of load. Using tools such as [Locust](https://locust.io/)
* Autoscaling could be considered also which can be implemented using the k8s `HorizontalPodAutoscaler` resource which can scale on CPU and memory targets. This may not be an adequate scaling target however, so custom metrics scaling can be acheived using [Keda](https://keda.sh/) which would allow scaling on requests
* Metrics. There are currently no metrics and the service is not configured to emit metrics. These can also be used to configure alerts and dashboards which can be used to monitor the application in production
* The manifests should not be applied manually using kubectl in production and a CD process should be used to ensure no single user has edit access to production enviroments. Examples of a CD solution could be ArgoCD, Github Actions, Flux etc
* The application should be configured to throttle the application on excessive requests to ensure the application cannot be overloaded and vulnerable to attacks such as DDoS
* Unit tests at a minimum in CI worklfow, further tests such as load tests and in a perfect world, chaos testing