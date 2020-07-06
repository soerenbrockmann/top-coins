# Top Coins - Coding Challenge

This service displays an up-to-date list of top assets and their current prices in USD.

## Data Sources

Cryptocompare API - to get the current ranking information for the top 100 assets.
Coinmarketcap API - to get the current USD prices

## Architecture

3 separate services (service oriented architecture):

Pricing Service - keeps the up-to-date pricing information
Ranking Service - keeps the up-to-date ranking information
HTTP-API Service - exposes a HTTP endpoint that returns the up-to-date list of up to 100 top coins prices.

### Patterns

- Messaging bus using RabbitMQ
- HTTP API
- Remote Procedure Calls

This service uses a combination of RabbitMQ and Remote Procedure Calls for internal service communication.
The reason for this choice is that the http-api-service can drop a request message to the message queue and the target service can simply reply with the requested data.

Furthermore, http requests are used to fetch data from the sources as well as for exposing the http-api-service.

## Setup

- Get API keys at the data providers.
- Add them to the pricing and ranking services

## Start Services

Run `docker-compose up --build`

## Fetch Ranking

`curl http://localhost:8080?limit=100`

Note: The ranking data source has a max limit of 100 items. Therefore, this is the maximum amount to get from the ranking.
